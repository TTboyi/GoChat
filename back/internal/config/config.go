package config

import (
	"chatapp/back/internal/model"
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/BurntSushi/toml"
	"github.com/redis/go-redis/v9"
)

// MainConfig 描述 HTTP 服务自身的基础信息。
type MainConfig struct {
	AppName string `toml:"appName"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}

// MysqlConfig 保存 MySQL 连接参数。
type MysqlConfig struct {
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	User         string `toml:"user"`
	Password     string `toml:"password"`
	DatabaseName string `toml:"databaseName"`
}

// RedisConfig 保存 Redis 连接参数。
type RedisConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Password string `toml:"password"`
	Db       int    `toml:"db"`
}

// AuthCodeConfig 预留给短信/验证码之类的第三方鉴权服务。
type AuthCodeConfig struct {
	AccessKeyID     string `toml:"accessKeyID"`
	AccessKeySecret string `toml:"accessKeySecret"`
	SignName        string `toml:"signName"`
	TemplateCode    string `toml:"templateCode"`
}

// LogConfig 保存日志输出路径。
type LogConfig struct {
	LogPath string `toml:"logPath"`
}

// KafkaConfig 描述消息队列相关配置。
type KafkaConfig struct {
	MessageMode string        `toml:"messageMode"`
	HostPort    string        `toml:"hostPort"`
	LoginTopic  string        `toml:"loginTopic"`
	LogoutTopic string        `toml:"logoutTopic"`
	ChatTopic   string        `toml:"chatTopic"`
	Partition   int           `toml:"partition"`
	Timeout     time.Duration `toml:"timeout"`
}

// StaticSrcConfig 描述静态文件上传目录。
type StaticSrcConfig struct {
	StaticAvatarPath string `toml:"staticAvatarPath"`
	StaticFilePath   string `toml:"staticFilePath"`
}

// Email 用于邮箱验证码登录流程。
type Email struct {
	SmtpHost string `toml:"smtp_host"`
	SmtpPort int    `toml:"smtp_port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// SecurityConfig 集中管理安全敏感配置，避免硬编码。
type SecurityConfig struct {
	// AllowedOrigins 是允许跨域访问的前端来源列表（CORS 和 WS CheckOrigin 共用）。
	AllowedOrigins []string `toml:"allowedOrigins"`
	// JWTSecret 是 HMAC 签名密钥，生产环境必须替换为随机强密钥。
	JWTSecret string `toml:"jwtSecret"`
	// JWTIssuer 是 token 颁发方标识。
	JWTIssuer string `toml:"jwtIssuer"`
	// TURNServer 是 TURN 中继服务器地址（host:port）。
	TURNServer string `toml:"turnServer"`
	// TURNSecret 是 coturn 的 static-auth-secret。
	TURNSecret string `toml:"turnSecret"`
}

// Config 是整个配置文件的聚合根。
// 读取 TOML 后，业务代码统一通过 GetConfig() 拿到它。
type Config struct {
	MainConfig      `toml:"mainConfig"`
	MysqlConfig     `toml:"mysqlConfig"`
	RedisConfig     `toml:"redisConfig"`
	AuthCodeConfig  `toml:"authCodeConfig"`
	LogConfig       `toml:"logConfig"`
	KafkaConfig     `toml:"kafkaConfig"`
	StaticSrcConfig `toml:"staticSrcConfig"`
	Email           `toml:"email"`
	SecurityConfig  `toml:"securityConfig"`
}

var config *Config = new(Config)

// LoadConfig 从约定路径读取 TOML 配置。
// 这里使用全局单例，是为了让启动阶段和业务层都能方便访问同一份配置。
func LoadConfig() error {
	if _, err := toml.DecodeFile("./back/internal/config/config.toml", config); err != nil {
		log.Fatal(err.Error())
		return err
	}
	return nil
}

// GetConfig 懒加载配置。
// 如果第一次调用时配置还没读入，就会自动触发 LoadConfig。
func GetConfig() *Config {
	if config == nil {
		config = new(Config)
		_ = LoadConfig()
	}
	return config
}

var db *gorm.DB

// InitDB 建立 GORM 连接，并在启动时自动迁移核心表结构。
// 这个项目把表迁移放在启动入口里，意味着“服务启动成功”通常也代表“数据库结构可用”。
func InitDB() {
	c := GetConfig().MysqlConfig

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.DatabaseName)

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// ✅ 在此处自动迁移你需要的表
	err = db.AutoMigrate(
		&model.UserInfo{},
		&model.GroupInfo{},
		&model.UserContact{},
		&model.ContactApply{},
		&model.Message{},
		&model.Session{},

		// 这里可以添加更多表，例如 &model.Message{} ...
	)
	if err != nil {
		log.Fatal("自动迁移失败:", err)
	}

	fmt.Println("✅ 成功连接数据库并自动迁移表")
}

// GetDB 返回全局数据库连接，必要时自动初始化。
func GetDB() *gorm.DB {
	if db == nil {
		InitDB()
	}
	return db
}

var rdb *redis.Client

// InitRedis 初始化 Redis 客户端并做一次 Ping。
// 这样可以在服务启动阶段尽早暴露配置错误，而不是等到业务请求进来才失败。
func InitRedis() {
	c := GetConfig().RedisConfig

	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		Password: c.Password,
		DB:       c.Db,
	})

	// 测试连接
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("连接 Redis 失败: ", err)
	}

	fmt.Println("✅ 成功连接 Redis")
}

// GetRedis 获取 Redis 客户端。
func GetRedis() *redis.Client {
	if rdb == nil {
		InitRedis()
	}
	return rdb
}
