package service

import (
	"log/slog"
	"strings"
	"time"

	"chatapp/back/internal/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedAdminUser 在服务启动时自动创建或更新管理员账号。
// username/password 来自配置文件；如果包含 "PLACEHOLDER" 占位符则跳过，
// 这样本地开发（config.toml 保持占位符）不会误创建账号，只有 VPS 部署后替换了真实值才生效。
func SeedAdminUser(db *gorm.DB, username, password string) {
	if strings.Contains(username, "PLACEHOLDER") || strings.Contains(password, "PLACEHOLDER") {
		slog.Info("管理员账号配置为占位符，跳过自动创建")
		return
	}
	if username == "" || password == "" {
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("管理员密码哈希失败", "err", err)
		return
	}

	var user model.UserInfo
	err = db.Where("nickname = ?", username).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		// 账号不存在，新建管理员
		raw := strings.ReplaceAll(uuid.NewString(), "-", "")
		newUser := model.UserInfo{
			Uuid:      raw,
			Nickname:  username,
			Telephone: "00000000000",
			Password:  string(hashed),
			IsAdmin:   1,
			Status:    0,
			CreatedAt: time.Now(),
		}
		if createErr := db.Create(&newUser).Error; createErr != nil {
			slog.Error("创建管理员账号失败", "err", createErr)
			return
		}
		slog.Info("管理员账号已创建", "nickname", username)
		return
	}
	if err != nil {
		slog.Error("查询管理员账号失败", "err", err)
		return
	}

	// 账号已存在：确保 is_admin=1 并同步最新密码（每次部署都会刷新）
	updates := map[string]interface{}{
		"is_admin": 1,
		"password": string(hashed),
	}
	if updateErr := db.Model(&user).Updates(updates).Error; updateErr != nil {
		slog.Error("更新管理员账号失败", "err", updateErr)
		return
	}
	slog.Info("管理员账号已同步", "nickname", username)
}
