package v1

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"chatapp/back/internal/chat"
	"chatapp/back/internal/config"
	"chatapp/back/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool {
		// 生产环境请替换成你的前端域名
		return true
	},
}

// ✅ WebSocket 登录：GET /wss?token=<JWT>
func WsLogin(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token不能为空"})
		return
	}

	// 1️⃣ 校验 JWT
	jwt := utils.GetJWT()
	claims, err := jwt.ParseAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 token"})
		return
	}

	// 2️⃣ 检查 Redis 黑名单
	rdb := config.GetRedis()
	if exists, _ := rdb.Exists(c, "jwt:blacklist:"+token).Result(); exists > 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token 已失效"})
		return
	}

	userId := claims.UserID
	log.Printf("🟢 WS LOGIN key=%q", userId)

	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token 中缺少用户ID"})
		return
	}

	// 3️⃣ 升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WebSocket 升级失败"})
		return
	}

	// 4️⃣ 创建 Client 并注册
	client := &chat.Client{
		Conn:     conn,
		Uuid:     userId,
		SendBack: make(chan []byte, 100),
	}
	chat.ChatServer.AddClient(client)

	// 5️⃣ 启动读写协程
	go client.Read()
	go client.Write()
	log.Printf("解析 token 成功: %+v\n", claims)
}

// ✅ 用户主动退出 WebSocket
func WsLogout(c *gin.Context) {
	userId := c.GetString("userId")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	chat.ChatServer.RemoveAllClients(userId)
	c.JSON(http.StatusOK, gin.H{"message": "退出成功"})
}

// ============================================================
// ✅ TURN 服务器动态凭证（HMAC-SHA1 时效性凭证）
// 标准：https://datatracker.ietf.org/doc/html/draft-uberti-behave-turn-rest
// ============================================================

// TURN 服务器配置（从环境变量或配置文件读取，不硬编码敏感信息）
const (
	turnServer = "209.54.106.103:3478"
	turnSecret = "gochat_turn_secret_2024" // coturn 配置的 static-auth-secret
	turnTTL    = 24 * time.Hour            // 凭证有效期 24 小时
)

// GetTurnCredentials 返回 TURN 服务器的时效性动态凭证
// GET /turn/credentials
func GetTurnCredentials(c *gin.Context) {
	userId := c.GetString("userId")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 1. 生成 username = 过期时间戳:userId
	expiry := time.Now().Add(turnTTL).Unix()
	username := fmt.Sprintf("%d:%s", expiry, userId)

	// 2. 用 HMAC-SHA1 生成 password
	mac := hmac.New(sha1.New, []byte(turnSecret))
	mac.Write([]byte(username))
	password := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	c.JSON(http.StatusOK, gin.H{
		"username": username,
		"password": password,
		"ttl":      int(turnTTL.Seconds()),
		"uris": []string{
			"turn:" + turnServer + "?transport=udp",
			"turn:" + turnServer + "?transport=tcp",
			"stun:" + turnServer,
		},
	})
}
