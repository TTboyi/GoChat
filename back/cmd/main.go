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

func main() {
	// 1) 配置 & 数据库
	//server := gin.Default()
	//server.Use(middleware.CORSMiddleware())

	if err := config.LoadConfig(); err != nil {
		log.Fatal("加载配置失败: ", err)
	}
	config.InitDB()

	// 2) 初始化全局 JWT（你现在用的方式）
	utils.InitJWT("chatapp-secret", "chatapp", 60, 1440)
	fmt.Println("JWT初始化成功:", string(utils.GetJWT().Key))
	// main.go
	cfg := config.GetConfig()
	utils.InitRedis(
		fmt.Sprintf("%s:%d", cfg.RedisConfig.Host, cfg.RedisConfig.Port),
		cfg.RedisConfig.Password,
		cfg.RedisConfig.Db,
	)

	//启动kafka
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
	chat.StartPersistConsumer(
		[]string{"127.0.0.1:9092"},
		"chat-persist-debug-1",
		"chat.message",
	)

	chat.StartCacheConsumer(
		[]string{"127.0.0.1:9092"},
		"chat-cache-debug-1",
		"chat.message",
	)

	// 3) 先启动 WebSocket Hub（用 goroutine，因为它是个死循环）
	// go chat.ChatServer.Run()

	// 4) 再启动 HTTP（阻塞）
	r := router.InitRouter() // 内部用 utils.GetJWT() 取全局 jwt
	if err := r.Run(":8000"); err != nil {
		log.Fatal("HTTP 启动失败: ", err)
	}

}
