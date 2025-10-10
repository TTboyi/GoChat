package v1

import (
	"log"
	"net/http"

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
	chat.ChatServer.Login <- client

	// 5️⃣ 启动读写协程
	go client.Read()
	go client.Write()
	log.Printf("解析 token 成功: %+v\n", claims)

}

// ✅ 用户主动退出 WebSocket
func WsLogout(c *gin.Context) {
	var form struct {
		UserId string `json:"userId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	chat.ChatServer.RemoveClient(form.UserId)
	c.JSON(http.StatusOK, gin.H{"message": "退出成功"})
}
