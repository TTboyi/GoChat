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

type MainConfig struct {
	AppName string `toml:"appName"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}

type MysqlConfig struct {
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	User         string `toml:"user"`
	Password     string `toml:"password"`
	DatabaseName string `toml:"databaseName"`
}

type RedisConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Password string `toml:"password"`
	Db       int    `toml:"db"`
}

type AuthCodeConfig struct {
	AccessKeyID     string `toml:"accessKeyID"`
	AccessKeySecret string `toml:"accessKeySecret"`
	SignName        string `toml:"signName"`
	TemplateCode    string `toml:"templateCode"`
}

type LogConfig struct {
	LogPath string `toml:"logPath"`
}

type KafkaConfig struct {
	MessageMode string        `toml:"messageMode"`
	HostPort    string        `toml:"hostPort"`
	LoginTopic  string        `toml:"loginTopic"`
	LogoutTopic string        `toml:"logoutTopic"`
	ChatTopic   string        `toml:"chatTopic"`
	Partition   int           `toml:"partition"`
	Timeout     time.Duration `toml:"timeout"`
}

type StaticSrcConfig struct {
	StaticAvatarPath string `toml:"staticAvatarPath"`
	StaticFilePath   string `toml:"staticFilePath"`
}

type Email struct {
	SmtpHost string `toml:"smtp_host"` // 对应 smtp_host = "..."
	SmtpPort int    `toml:"smtp_port"` // 对应 smtp_port = 465
	Username string `toml:"username"`  // 对应 username = "..."
	Password string `toml:"password"`  // 对应 password = "..."
}

type Config struct {
	MainConfig      `toml:"mainConfig"`
	MysqlConfig     `toml:"mysqlConfig"`
	RedisConfig     `toml:"redisConfig"`
	AuthCodeConfig  `toml:"authCodeConfig"`
	LogConfig       `toml:"logConfig"`
	KafkaConfig     `toml:"kafkaConfig"`
	StaticSrcConfig `toml:"staticSrcConfig"`
	Email           `toml:"email"`
}

var config *Config = new(Config)

func LoadConfig() error {
	if _, err := toml.DecodeFile("./back/internal/config/config.toml", config); err != nil {
		log.Fatal(err.Error())
		return err
	}
	return nil
}

func GetConfig() *Config {
	if config == nil {
		config = new(Config)
		_ = LoadConfig()
	}
	return config
}

var db *gorm.DB

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

func GetDB() *gorm.DB {
	if db == nil {
		InitDB()
	}
	return db
}

var rdb *redis.Client

// InitRedis 初始化 Redis 客户端
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

// GetRedis 获取 Redis 客户端
func GetRedis() *redis.Client {
	if rdb == nil {
		InitRedis()
	}
	return rdb
}
