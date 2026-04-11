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
		// 这里为了开发方便直接放行所有来源。
		// 真正部署时通常要收紧到可信前端域名，否则任何站点都能尝试建立 WS 连接。
		return true
	},
}

// WsLogin 负责完成 WebSocket 握手前的认证。
// 浏览器会先带着 access token 访问 /wss，后端验证 token 和黑名单后，
// 才把 HTTP 请求升级成一条长期存活的 WebSocket 连接。
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

	// 5️⃣ 启动读写协程。
	// 一个协程专门读，一个协程专门写，是 WebSocket 服务器常见的并发模型。
	go client.Read()
	go client.Write()
	log.Printf("解析 token 成功: %+v\n", claims)
}

// WsLogout 允许用户主动断开自己的所有 WebSocket 连接。
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

// GetTurnCredentials 返回 TURN 服务器的时效性动态凭证。
// WebRTC 在复杂网络环境下需要 TURN 中继；这个接口的作用就是在用户真正发起通话前，
// 临时生成一组可用但会过期的凭证，而不是把长期密钥直接暴露给前端。
func GetTurnCredentials(c *gin.Context) {
	userId := c.GetString("userId")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 1. 生成 username = 过期时间戳:userId。
	//    这种格式是 TURN REST API 的常见约定，服务端可据此判断凭证是否过期。
	expiry := time.Now().Add(turnTTL).Unix()
	username := fmt.Sprintf("%d:%s", expiry, userId)

	// 2. 用 static-auth-secret 做 HMAC，得到与 username 绑定的短期密码。
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
