// ============================================================
// 文件：back/internal/service/user_service.go
// 作用：用户注册、登录、信息查询、信息更新的业务逻辑。
//
// 密码安全：
//   RegisterUser：用 bcrypt.DefaultCost（当前为 10 轮）生成哈希，
//     同样的密码每次哈希结果不同（因为有随机 salt），无法通过彩虹表攻击。
//   LoginUser：支持新旧两种密码格式：
//     - bcrypt 哈希（新注册账号）：用 CompareHashAndPassword 比对
//     - 明文密码（迁移前的旧账号）：直接字符串比较
//     旧账号首次登录成功后自动升级为 bcrypt 哈希（"密码迁移"）
//
// 用户信息缓存策略（GetUserInfo）：
//   1. 先查 Redis（key = "user_info_v2:{userId}"，TTL = 15分钟）
//   2. 缓存命中直接返回，不查数据库
//   3. 缓存未命中则查 MySQL，写入 Redis 后返回
//   UpdateUserInfo 更新成功后主动删除 Redis 缓存（Cache-Aside 模式的失效策略），
//   下次读取时重新从 DB 加载最新数据。
//
// FindOrCreateUserByEmail：
//   邮箱验证码登录场景：如果邮箱从未注册过，自动创建账号。
//   昵称取邮箱 @ 前面的部分（如 hello@gmail.com → 昵称 hello）。
// ============================================================
package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"chatapp/back/internal/config"
	"chatapp/back/internal/dao"
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/dto/resp"
	"chatapp/back/internal/model"
	"chatapp/back/utils"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const userInfoCachePrefix = "user_info_v2:"
const userInfoCacheTTL = 15 * time.Minute

func RegisterUser(db *gorm.DB, user *model.UserInfo) error {
	existingUser, err := dao.GetUserByName(db, user.Nickname)
	if err == nil && existingUser != nil {
		return errors.New("用户名已注册")
	}

	// 哈希密码
	if user.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return errors.New("密码加密失败")
		}
		user.Password = string(hashed)
	}

	user.CreatedAt = time.Now()

	return dao.CreateUser(db, user)
}

func LoginUser(db *gorm.DB, name, password string, jwt *utils.ARJWT) (string, string, error) {
	user, err := dao.GetUserByName(db, name)
	if err != nil || user == nil {
		return "", "", errors.New("用户不存在")
	}

	// 先尝试 bcrypt 比较（新注册/已迁移账号）
	if bcryptErr := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); bcryptErr != nil {
		// 回退到明文比较，兼容迁移前的旧账号
		if user.Password != password {
			return "", "", errors.New("密码错误")
		}
		// 旧账号登录成功 → 自动将明文密码升级为 bcrypt hash
		if hashed, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost); hashErr == nil {
			db.Model(user).Update("password", string(hashed))
		}
	}

	access, refresh, err := jwt.GenerateToken(user.Uuid, user.IsAdmin)
	if err != nil {
		return "", "", errors.New("生成令牌失败")
	}

	return access, refresh, nil
}

func FindOrCreateUserByEmail(email string) (*model.UserInfo, error) {
	db := config.GetDB()

	var user model.UserInfo
	if err := db.Where("email = ?", email).First(&user).Error; err == nil {
		return &user, nil
	}

	// 生成 <= 20 位 uuid（去掉横杠更紧凑）
	raw := strings.ReplaceAll(uuid.NewString(), "-", "")
	if len(raw) > 20 {
		raw = raw[:20]
	}

	nickname := email
	if at := strings.Index(email, "@"); at > 0 {
		nickname = email[:at]
	}

	newUser := model.UserInfo{
		Uuid:      raw,
		Nickname:  nickname,
		Telephone: "", // 邮箱登录不强制电话
		Email:     email,
		Avatar:    "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png",
		Gender:    0,
		Signature: "",
		Password:  "",
		Birthday:  "",
		CreatedAt: time.Now(),
		IsAdmin:   0,
		Status:    0,
	}

	if err := db.Create(&newUser).Error; err != nil {
		return nil, errors.New("创建用户失败")
	}
	return &newUser, nil
}

func GetUserInfo(userId string) (*resp.UserInfoResponse, error) {
	rdb := config.GetRedis()
	ctx := context.Background()
	cacheKey := userInfoCachePrefix + userId

	// 优先读缓存
	if cached, err := rdb.Get(ctx, cacheKey).Result(); err == nil {
		var respUser resp.UserInfoResponse
		if json.Unmarshal([]byte(cached), &respUser) == nil {
			return &respUser, nil
		}
	}

	db := config.GetDB()
	var user model.UserInfo
	if err := db.Where("uuid = ?", userId).First(&user).Error; err != nil {
		return nil, err
	}

	respUser := &resp.UserInfoResponse{
		Uuid:      user.Uuid,
		Nickname:  user.Nickname,
		Telephone: user.Telephone,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Signature: user.Signature,
		IsAdmin:   user.IsAdmin,
	}

	// 写入缓存，忽略错误
	if data, err := json.Marshal(respUser); err == nil {
		_ = rdb.Set(ctx, cacheKey, data, userInfoCacheTTL).Err()
	}

	return respUser, nil
}

// UpdateUserInfo 更新用户资料
func UpdateUserInfo(userId string, form *req.UpdateUserRequest) (*resp.UserInfoResponse, error) {
	db := config.GetDB()

	var user model.UserInfo
	if err := db.Where("uuid = ?", userId).First(&user).Error; err != nil {
		return nil, err
	}

	// 修改字段
	if form.Nickname != "" {
		user.Nickname = form.Nickname
	}
	if form.Email != "" {
		user.Email = form.Email
	}
	if form.Avatar != "" {
		user.Avatar = form.Avatar
	}
	if form.Signature != "" {
		user.Signature = form.Signature
	}
	if form.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashed)
	}

	// 保存
	if err := db.Save(&user).Error; err != nil {
		return nil, err
	}

	// 更新成功后删除缓存，下次读取时重新从DB加载
	rdb := config.GetRedis()
	_ = rdb.Del(context.Background(), userInfoCachePrefix+userId).Err()

	// 返回安全 DTO
	respUser := &resp.UserInfoResponse{
		Uuid:      user.Uuid,
		Nickname:  user.Nickname,
		Telephone: user.Telephone,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Signature: user.Signature,
		IsAdmin:   user.IsAdmin,
	}
	return respUser, nil
}
