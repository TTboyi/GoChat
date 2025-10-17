package chat

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"chatapp/back/internal/config"
	"chatapp/back/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ========= 供 client.go / ws.go 共用的消息结构 =========

// 前端发来的消息（点对点 & 群聊通用）
type ChatEnvelope struct {
	Type      int8   `json:"type"`              // 0=文本, 1=文件, 2=通话（可扩展）
	Content   string `json:"content,omitempty"` // 文本内容
	Url       string `json:"url,omitempty"`     // 文件/图片等的 URL
	FileName  string `json:"fileName,omitempty"`
	FileType  string `json:"fileType,omitempty"`
	FileSize  string `json:"fileSize,omitempty"`
	SendId    string `json:"sendId"`    // 发送者 UUID（若前端不传，后端用连接的 client.Uuid 兜底）
	ReceiveId string `json:"receiveId"` // 接收方 UUID（用户或群）
}

// 推送给前端的消息
type OutgoingMessage struct {
	Uuid      string `json:"uuid"` // 消息ID
	Type      int8   `json:"type"`
	Content   string `json:"content,omitempty"`
	Url       string `json:"url,omitempty"`
	SendId    string `json:"sendId"`
	ReceiveId string `json:"receiveId"`
	CreatedAt int64  `json:"createdAt"` // Unix 秒
}

// ======================================================

// Server 聊天主机
type Server struct {
	Clients  map[string]*Client // 在线用户：userUuid -> *Client
	Mutex    *sync.Mutex
	Transmit chan ChatEnvelope // 消息入口（client.Read()->这里）
	Login    chan *Client
	Logout   chan *Client
}

// ✅ 记录每个群在线成员
var groupMembers = make(map[string]map[string]bool) // groupId -> userId -> 在线状态

// 全局唯一
var ChatServer = &Server{
	Clients:  make(map[string]*Client),
	Mutex:    &sync.Mutex{},
	Transmit: make(chan ChatEnvelope, 1024),
	Login:    make(chan *Client, 128),
	Logout:   make(chan *Client, 128),
}

func (s *Server) AddUserToGroup(userId, groupId string) {
	if _, ok := groupMembers[groupId]; !ok {
		groupMembers[groupId] = make(map[string]bool)
	}
	groupMembers[groupId][userId] = true
	log.Printf("✅ 用户 %s 已加入群订阅 %s", userId, groupId)
}

// Run 启动消息循环
func (s *Server) Run() {
	for {
		select {
		// 新客户端登录
		case client := <-s.Login:
			s.Mutex.Lock()
			s.Clients[client.Uuid] = client
			s.Mutex.Unlock()

			log.Printf("用户 %s 登录\n", client.Uuid)
			// 给用户一个欢迎消息
			msg := fmt.Sprintf("欢迎用户 %s 加入聊天室", client.Uuid)
			client.SendBack <- []byte(msg)

		// 客户端退出
		case client := <-s.Logout:
			s.Mutex.Lock()
			delete(s.Clients, client.Uuid)
			s.Mutex.Unlock()

			log.Printf("用户 %s 退出\n", client.Uuid)
			_ = client.Conn.Close()

			// 收到消息
		case env := <-s.Transmit:
			// 统一走路由 + 入库 + 下发（点对点/群聊都在这里处理）
			if err := s.routeAndPersist(env); err != nil {
				log.Printf("❌ routeAndPersist 失败: %v", err)
			}

		}
	}
}

// ============== 核心：路由 & 存库 ==============

func (s *Server) routeAndPersist(env ChatEnvelope) error {
	db := config.GetDB()

	// 兜底：如果前端没带 sendId，用连接ID（client 在 Read 前已写入）
	if env.SendId == "" {
		return fmt.Errorf("empty sendId")
	}

	// 不同收件：用户 or 群
	if isGroup(env.ReceiveId) {
		return s.handleGroup(db, env)
	}
	return s.handleDirect(db, env)
}

func isGroup(id string) bool {
	return strings.HasPrefix(id, "G") // 你的群UUID约定：以 "G" 开头
}

// 处理点对点
func (s *Server) handleDirect(db *gorm.DB, env ChatEnvelope) error {
	// 1) 确保会话（发起人->对方）
	sessID, recvName, recvAvatar, err := ensureSessionForDirect(db, env.SendId, env.ReceiveId)
	if err != nil {
		return err
	}

	// 2) 构建 & 入库
	msgID := newIDWithPrefix("M")
	now := time.Now()
	out := OutgoingMessage{
		Uuid:      msgID,
		Type:      env.Type,
		Content:   env.Content,
		Url:       env.Url,
		SendId:    env.SendId,
		ReceiveId: env.ReceiveId,
		CreatedAt: now.Unix(),
	}

	senderName, senderAvatar, _ := loadUserBasic(db, env.SendId)

	msg := model.Message{
		Uuid:       msgID,
		SessionId:  sessID,
		Type:       env.Type,
		Content:    env.Content,
		Url:        env.Url,
		SendId:     env.SendId,
		SendName:   nz(senderName, "用户"),
		SendAvatar: nz(senderAvatar, "default_avatar.png"),
		ReceiveId:  env.ReceiveId,
		FileType:   env.FileType,
		FileName:   env.FileName,
		FileSize:   env.FileSize,
		Status:     0,
		CreatedAt:  now,
	}

	if err := db.Create(&msg).Error; err != nil {
		return fmt.Errorf("保存消息失败: %w", err)
	}

	// 3) 下发：对方 & 自己（回显）
	raw, _ := json.Marshal(out)
	s.deliverToUser(env.ReceiveId, raw) // 对方在线才会收到
	s.deliverToUser(env.SendId, raw)    // 自己回显
	// 成功下发至少一份即可视为“已发送”
	_ = db.Model(&model.Message{}).Where("uuid = ?", msgID).Update("status", 1).Error

	_ = recvName
	_ = recvAvatar
	return nil
}

// 处理群聊
func (s *Server) handleGroup(db *gorm.DB, env ChatEnvelope) error {
	var group model.GroupInfo
	if err := db.Where("uuid = ?", env.ReceiveId).First(&group).Error; err != nil {
		return fmt.Errorf("群聊不存在")
	}

	// 1) 先为“发送者->该群”确保会话（message.session_id 非空）
	sessID, _, _, err := ensureSessionForGroup(db, env.SendId, &group)
	if err != nil {
		return err
	}

	// 2) 入库
	msgID := newIDWithPrefix("M")
	now := time.Now()
	out := OutgoingMessage{
		Uuid:      msgID,
		Type:      env.Type,
		Content:   env.Content,
		Url:       env.Url,
		SendId:    env.SendId,
		ReceiveId: env.ReceiveId, // 群ID
		CreatedAt: now.Unix(),
	}

	senderName, senderAvatar, _ := loadUserBasic(db, env.SendId)

	msg := model.Message{
		Uuid:       msgID,
		SessionId:  sessID, // 以“发送者->群”的 session 兜底
		Type:       env.Type,
		Content:    env.Content,
		Url:        env.Url,
		SendId:     env.SendId,
		SendName:   nz(senderName, "用户"),
		SendAvatar: nz(senderAvatar, "default_avatar.png"),
		ReceiveId:  env.ReceiveId, // 群ID
		FileType:   env.FileType,
		FileName:   env.FileName,
		FileSize:   env.FileSize,
		Status:     0,
		CreatedAt:  now,
	}
	if err := db.Create(&msg).Error; err != nil {
		return fmt.Errorf("保存群聊消息失败: %w", err)
	}

	// 3) 群广播（仅在线）
	var members []string
	_ = json.Unmarshal(group.Members, &members)
	raw, _ := json.Marshal(out)
	for _, uid := range members {
		s.deliverToUser(uid, raw)
	}
	_ = db.Model(&model.Message{}).Where("uuid = ?", msgID).Update("status", 1).Error

	return nil
}

// 推送给在线用户
func (s *Server) deliverToUser(userId string, raw []byte) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	if c, ok := s.Clients[userId]; ok {
		select {
		case c.SendBack <- raw:
		default:
			// 客户端下游拥堵，防止阻塞
			log.Printf("用户 %s 下行拥堵，消息丢弃\n", userId)
		}
	}
}

// ============== 会话兜底 & 基础查询 ==============

func ensureSessionForDirect(db *gorm.DB, sendId, recvId string) (sessionUuid, recvName, recvAvatar string, err error) {
	// 先查是否已有
	var sess model.Session
	if err = db.Where("send_id = ? AND receive_id = ?", sendId, recvId).First(&sess).Error; err == nil {
		return sess.Uuid, sess.ReceiveName, sess.Avatar, nil
	}

	// 没有则创建
	recvName, recvAvatar, _ = loadUserBasic(db, recvId)
	sess = model.Session{
		Uuid:        newIDWithPrefix("S"),
		SendId:      sendId,
		ReceiveId:   recvId,
		ReceiveName: nz(recvName, "用户"),
		Avatar:      nz(recvAvatar, "default_avatar.png"),
		CreatedAt:   time.Now(),
	}
	if err = db.Create(&sess).Error; err != nil {
		return "", "", "", fmt.Errorf("创建会话失败: %w", err)
	}
	return sess.Uuid, sess.ReceiveName, sess.Avatar, nil
}

func ensureSessionForGroup(db *gorm.DB, sendId string, group *model.GroupInfo) (sessionUuid, groupName, groupAvatar string, err error) {
	// 先查
	var sess model.Session
	if err = db.Where("send_id = ? AND receive_id = ?", sendId, group.Uuid).First(&sess).Error; err == nil {
		return sess.Uuid, sess.ReceiveName, sess.Avatar, nil
	}
	// 创建
	sess = model.Session{
		Uuid:        newIDWithPrefix("S"),
		SendId:      sendId,
		ReceiveId:   group.Uuid,
		ReceiveName: nz(group.Name, "群聊"),
		Avatar:      nz(group.Avatar, "default_avatar.png"),
		CreatedAt:   time.Now(),
	}
	if err = db.Create(&sess).Error; err != nil {
		return "", "", "", fmt.Errorf("创建群会话失败: %w", err)
	}
	return sess.Uuid, sess.ReceiveName, sess.Avatar, nil
}

func loadUserBasic(db *gorm.DB, userId string) (nickname, avatar string, err error) {
	var u model.UserInfo
	if e := db.Where("uuid = ?", userId).First(&u).Error; e != nil {
		return "", "", e
	}
	return u.Nickname, u.Avatar, nil
}

// ============== 工具 ==============

func newIDWithPrefix(p string) string {
	// 生成 19 位随机（去掉 - 的 uuid），再加 1 位前缀，正好 20
	raw := strings.ReplaceAll(uuid.New().String(), "-", "")
	return p + raw[:19]
}

func nz(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// ✅ 新增：添加客户端
func (s *Server) AddClient(c *Client) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.Clients[c.Uuid] = c
	log.Printf("✅ 用户 %s 登录", c.Uuid)
	c.SendBack <- []byte(fmt.Sprintf("欢迎用户 %s 加入聊天室", c.Uuid))
}

// ✅ 新增：移除客户端
func (s *Server) RemoveClient(userId string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if c, ok := s.Clients[userId]; ok {
		_ = c.Conn.Close()
		delete(s.Clients, userId)
		log.Printf("❎ 用户 %s 退出", userId)
	}
}
