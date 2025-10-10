package service

import (
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

func RegisterUser(db *gorm.DB, user *model.UserInfo) error {
	existingUser, err := dao.GetUserByName(db, user.Nickname)
	if err == nil && existingUser != nil {
		return errors.New("手机号已注册")
	}

	user.CreatedAt = time.Now()

	return dao.CreateUser(db, user)
}

func LoginUser(db *gorm.DB, name, password string, jwt *utils.ARJWT) (string, string, error) {
	user, err := dao.GetUserByName(db, name)
	if err != nil || user == nil {
		return "", "", errors.New("用户不存在")
	}

	if user.Password != password {
		return "", "", errors.New("密码错误")
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
	db := config.GetDB()

	var user model.UserInfo
	if err := db.Where("uuid = ?", userId).First(&user).Error; err != nil {
		return nil, err
	}

	// 转换成安全的响应 DTO
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
