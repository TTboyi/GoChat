package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/model"
	"chatapp/back/utils"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const EmailCaptchaPrefix = "email_captcha:"

// 发送邮箱验证码
func SendEmailCaptcha(email string) error {
	rdb := config.GetRedis()

	// 检查是否有发送频率限制
	key := EmailCaptchaPrefix + email
	exists, _ := rdb.Exists(context.Background(), key).Result()
	if exists == 1 {
		return errors.New("请稍后再试，验证码已发送")
	}

	// 生成验证码
	code := utils.GenerateEmailCode()

	// 存入 Redis，过期时间 5 分钟
	err := rdb.Set(context.Background(), key, code, 5*time.Minute).Err()
	if err != nil {
		return err
	}

	// 发送邮件
	subject := "【ChatApp】邮箱验证码"
	body := fmt.Sprintf("您的验证码是：%s，有效期 5 分钟。", code)
	return utils.SendEmail(email, subject, body)
}

// 邮箱验证码登录
func EmailCaptchaLogin(email, code string) (string, string, error) {
	rdb := config.GetRedis()
	key := EmailCaptchaPrefix + email

	// 从 Redis 获取验证码
	storedCode, err := rdb.Get(context.Background(), key).Result()
	if err != nil {
		return "", "", errors.New("验证码不存在或已过期")
	}
	if storedCode != code {
		return "", "", errors.New("验证码错误")
	}

	db := config.GetDB()
	var user model.UserInfo

	// 查找用户是否存在
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户不存在 → 注册新账号
			user = model.UserInfo{
				Uuid:     "U" + uuid.NewString()[:19],
				Nickname: "用户_" + email,
				Email:    email,
				Password: "", // 邮箱登录用户初始无密码
				IsAdmin:  0,
				Status:   0,
				Avatar:   "default_avatar.png",
			}
			if err := db.Create(&user).Error; err != nil {
				return "", "", errors.New("用户创建失败")
			}
		} else {
			return "", "", errors.New("数据库错误")
		}
	}

	// 删除验证码（避免重复使用）
	_ = rdb.Del(context.Background(), key).Err()

	// 生成 JWT
	jwt := utils.GetJWT()
	access, refresh, err := jwt.GenerateToken(user.Uuid, user.IsAdmin)
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
}
