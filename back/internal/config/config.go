// ============================================================
// 文件：back/internal/config/config.go
// 作用：定义整个后端的配置结构体，并提供"读取 TOML 文件"和
//       "初始化数据库、Redis"的函数。
//
// 设计思路：
//   所有配置都聚合在一个顶层结构体 Config 中，用 TOML 格式文件存储。
//   TOML 是一种配置文件格式，比 JSON 更易读，比 YAML 更严格。
//   config.toml 里的 [区块名] 对应 Config 里的字段名（通过 toml:"xxx" 标签映射）。
//
// 单例模式：
//   db、rdb（Redis客户端）、config 都是包级别的全局变量，
//   整个程序运行期间只有一份实例，所有包都共享同一个连接。
//   这样做的好处是避免重复建立连接（每次建连都有开销），
//   缺点是测试时需要注意隔离。
//
// 关键概念：
//   GORM   - Go 语言的 ORM 库（对象关系映射），让你用 Go 结构体操作数据库，
//             而不必直接写 SQL 语句。`gorm:"..."` 标签定义了字段在数据库中的行为。
//   Redis  - 一个运行在内存中的键值对数据库，读写速度极快（微秒级），
//             但重启后数据会丢失（除非配置持久化）。
//   AutoMigrate - GORM 自动对比模型结构体与数据库表，自动 ADD COLUMN / CREATE TABLE，
//                 但不会自动 DROP COLUMN（安全起见）。
// ============================================================

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

// LogConfig 保存日志文件输出配置。
type LogConfig struct {
	// LogPath 是日志文件路径，留空则只写 stdout。
	LogPath string `toml:"logPath"`
	// MaxSizeMB 单个日志文件最大大小（MB），超出后轮转。
	MaxSizeMB int `toml:"maxSizeMB"`
	// MaxAgeDays 保留最近几天的旧日志文件。
	MaxAgeDays int `toml:"maxAgeDays"`
	// MaxBackups 最多保留几个旧日志文件。
	MaxBackups int `toml:"maxBackups"`
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

// AdminConfig 描述启动时自动创建/更新的管理员账号。
// 占位符（含 PLACEHOLDER 字样）时跳过，只在 VPS 部署后生效。
type AdminConfig struct {
	// Username 管理员昵称，也作为登录账号。
	Username string `toml:"username"`
	// Password 管理员明文密码，启动时自动 bcrypt 哈希后存储。
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
	AdminConfig     `toml:"adminConfig"`
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
