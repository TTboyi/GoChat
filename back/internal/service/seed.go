// ============================================================
// 文件：back/internal/service/seed.go
// 作用：初始化（种子化）管理员账号，在服务启动时自动创建/更新管理员用户。
//
// "Seed"（种子）是后端开发中的常见术语，指"初始化一些必要的基础数据"。
//
// SeedAdminUser 的执行逻辑：
//   1. 如果 username 包含 "PLACEHOLDER" 字样，跳过（本地开发环境无需创建）
//   2. 查找数据库中是否已有这个昵称的用户
//   3. 如果不存在，创建新管理员用户（is_admin=1）
//   4. 如果存在但密码不一致，更新密码（支持"修改配置文件密码后重启服务"的运维场景）
//   5. 密码始终以 bcrypt 哈希形式存储
//
// 这个机制非常适合自动化部署：CI/CD 流程可以在部署时把配置文件里的
// ADMIN_USERNAME_PLACEHOLDER 替换为真实用户名，服务启动时自动建立管理员账号。
// ============================================================
package service

import (
	"errors"
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
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 账号不存在，新建管理员
		// 截取前20位保证与 char(20) 列类型一致
		raw := strings.ReplaceAll(uuid.NewString(), "-", "")
		if len(raw) > 20 {
			raw = raw[:20]
		}
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
