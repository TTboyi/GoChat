package main

import (
	"fmt"
	"log"

	"chatapp/back/internal/chat"
	"chatapp/back/internal/config"
	"chatapp/back/internal/router"
	"chatapp/back/utils"
	//"chatapp/back/internal/middleware" // 替换成你项目中间件的真实路径
	//"github.com/gin-gonic/gin"
)

// main 负责把“可运行的聊天系统”组装起来。
// 对学习这个项目的人来说，这个入口文件最值得关注的是启动顺序：
// 1. 先读取配置并建立数据库/Redis 连接；
// 2. 再初始化 JWT 和 Kafka 消息管道；
// 3. 启动后台清理任务；
// 4. 最后启动 Gin HTTP 服务，对外提供 REST 和 WebSocket 接口。
func main() {
	// 1) 配置 & 数据库：后面所有模块都会依赖全局配置和 DB 单例。
	//server := gin.Default()
	//server.Use(middleware.CORSMiddleware())

	if err := config.LoadConfig(); err != nil {
		log.Fatal("加载配置失败: ", err)
	}
	config.InitDB()

	// 2) 初始化全局 JWT：HTTP 鉴权、刷新 token、WebSocket 握手都会复用它。
	utils.InitJWT("chatapp-secret", "chatapp", 60, 1440)
	log.Println("JWT初始化成功")

	// 3) 初始化 Redis：主要用于 token 黑名单、验证码、缓存等场景。
	cfg := config.GetConfig()
	utils.InitRedis(
		fmt.Sprintf("%s:%d", cfg.RedisConfig.Host, cfg.RedisConfig.Port),
		cfg.RedisConfig.Password,
		cfg.RedisConfig.Db,
	)

	// 4) 启动 Kafka 生产者和三个消费者。
	//    这里体现了项目的消息主链路：WebSocket 收到消息后先写入 Kafka，
	//    再由不同消费者分别负责“实时分发 / 持久化 / 缓存更新”。
	err := chat.InitKafkaProducer(
		[]string{"127.0.0.1:9092"},
		"chat.message",
	)

	chat.StartDispatcherConsumer(
		[]string{"127.0.0.1:9092"},
		"chat-dispatcher-debug-1",
		"chat.message",
	)

	if err != nil {
		log.Fatalf("Kafka init failed: %v", err)
	}

	// Persist Consumer：把消息真正写入 MySQL，保证历史消息可追溯。
	chat.StartPersistConsumer(
		[]string{"127.0.0.1:9092"},
		"chat-persist-debug-1",
		"chat.message",
	)

	// Cache Consumer：把部分会话/消息状态同步到缓存层，提高读取效率。
	chat.StartCacheConsumer(
		[]string{"127.0.0.1:9092"},
		"chat-cache-debug-1",
		"chat.message",
	)

	// 5) 旧版内存 Hub 的 Run 循环目前没有显式启动，
	//    因为当前主链路已经改成“Client -> Kafka -> Consumers”。
	//    这段注释保留了项目演进的痕迹，方便理解两套方案的差异。
	// go chat.ChatServer.Run()

	// 6) 启动后台文件清理任务，定期处理上传目录中的过期文件。
	chat.StartFileCleanup(config.GetConfig().StaticFilePath)

	// 7) 最后启动 HTTP 服务。
	//    InitRouter 会注册 REST 接口、静态资源、WebSocket 登录入口等全部路由。
	r := router.InitRouter() // 内部用 utils.GetJWT() 取全局 jwt
	if err := r.Run(":8000"); err != nil {
		log.Fatal("HTTP 启动失败: ", err)
	}

}
