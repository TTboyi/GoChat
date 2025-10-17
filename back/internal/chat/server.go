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
	Uuid       string `json:"uuid"` // 消息ID
	Type       int8   `json:"type"`
	Content    string `json:"content,omitempty"`
	Url        string `json:"url,omitempty"`
	SendId     string `json:"sendId"`
	SendName   string `json:"sendName"`   // ✅ 新增
	SendAvatar string `json:"sendAvatar"` // ✅ 新增
	ReceiveId  string `json:"receiveId"`
	CreatedAt  int64  `json:"createdAt"` // Unix 秒
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

		case env := <-s.Transmit:
			// ✅ 系统消息不入库，直接广播
			if env.Type == 99 {
				raw, _ := json.Marshal(env)
				for userId := range s.Clients {
					s.deliverToUser(userId, raw)
				}
				continue
			}

		}

	}
}

// ============== 核心：路由 & 存库 ==============

func (s *Server) routeAndPersist(env ChatEnvelope) error {
	db := config.GetDB()

	if env.SendId == "" {
		return fmt.Errorf("empty sendId")
	}
	// ★ 用新的群判断
	if s.isGroupTarget(db, env.ReceiveId) {
		return s.handleGroup(db, env)
	}
	return s.handleDirect(db, env)
}

// 用“订阅表”或数据库判断是否为群
func (s *Server) isGroupTarget(db *gorm.DB, id string) bool {
	if id == "" {
		return false
	}
	// ① 快速判断：有订阅（有人 join 过）就认为是群
	if _, ok := groupMembers[id]; ok {
		return true
	}
	// ② 数据库兜底判断
	var cnt int64
	if err := db.Model(&model.GroupInfo{}).Where("uuid = ?", id).Count(&cnt).Error; err == nil && cnt > 0 {
		return true
	}
	return false
}

// 处理点对点
func (s *Server) handleDirect(db *gorm.DB, env ChatEnvelope) error {
	// 1) 确保会话（发起人->对方）
	sessID, recvName, recvAvatar, err := ensureSessionForDirect(db, env.SendId, env.ReceiveId)
	if err != nil {
		return err
	}
	senderName, senderAvatar, _ := loadUserBasic(db, env.SendId)
	// 2) 构建 & 入库
	msgID := newIDWithPrefix("M")
	now := time.Now()
	out := OutgoingMessage{
		Uuid:       msgID,
		Type:       env.Type,
		Content:    env.Content,
		Url:        env.Url,
		SendId:     env.SendId,
		SendName:   nz(senderName, "用户"),        // ✅
		SendAvatar: nz(senderAvatar, "default"), // ✅
		ReceiveId:  env.ReceiveId,
		CreatedAt:  now.Unix(),
	}

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

	// 确保会话
	sessID, _, _, err := ensureSessionForGroup(db, env.SendId, &group)
	if err != nil {
		return err
	}

	// 入库
	msgID := newIDWithPrefix("M")
	now := time.Now()
	senderName, senderAvatar, _ := loadUserBasic(db, env.SendId)
	out := OutgoingMessage{
		Uuid:       msgID,
		Type:       env.Type,
		Content:    env.Content,
		Url:        env.Url,
		SendId:     env.SendId,
		SendName:   nz(senderName, "用户"),
		SendAvatar: nz(senderAvatar, "default"),
		ReceiveId:  env.ReceiveId, // 群ID
		CreatedAt:  now.Unix(),
	}

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
		return fmt.Errorf("保存群消息失败: %w", err)
	}

	// ✅ 使用订阅用户推送（而不是 group.Members）
	raw, _ := json.Marshal(out)
	if subs, ok := groupMembers[env.ReceiveId]; ok {
		for userId := range subs {
			s.deliverToUser(userId, raw)
		}
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

// ✅ 推送“群已解散”通知
func (s *Server) PushGroupDismiss(groupId string) {
	raw, _ := json.Marshal(map[string]interface{}{
		"uuid":      newIDWithPrefix("SYS"),
		"type":      99, // ✅ 自定义类型：99 表示系统事件
		"receiveId": groupId,
		"content":   "group_dismiss", // ✅ 事件内容
		"createdAt": time.Now().Unix(),
	})

	// ✅ 推给这个群的所有在线成员
	if subs, ok := groupMembers[groupId]; ok {
		for uid := range subs {
			s.deliverToUser(uid, raw)
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

// // ✅ 新增：移除客户端
// func (s *Server) RemoveClient(userId string) {
// 	s.Mutex.Lock()
// 	defer s.Mutex.Unlock()

// 	if c, ok := s.Clients[userId]; ok {
// 		_ = c.Conn.Close()
// 		delete(s.Clients, userId)
// 		log.Printf("❎ 用户 %s 退出", userId)
// 	}
// }

// 在 server.go 里加一个工具函数
func (s *Server) removeUserFromAllGroups(userId string) {
	for gid, subs := range groupMembers {
		if subs[userId] {
			delete(subs, userId)
			if len(subs) == 0 {
				delete(groupMembers, gid)
			}
		}
	}
}

// 在 RemoveClient 里调用
func (s *Server) RemoveClient(userId string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if c, ok := s.Clients[userId]; ok {
		_ = c.Conn.Close()
		delete(s.Clients, userId)
		log.Printf("❎ 用户 %s 退出", userId)
	}
	s.removeUserFromAllGroups(userId) // ✅ 清理订阅，防止脏数据
}

// DeliverToUser 导出版（供其它包推送控制消息用）
func (s *Server) DeliverToUser(userId string, raw []byte) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	if c, ok := s.Clients[userId]; ok {
		select {
		case c.SendBack <- raw:
		default:
			// 下行拥塞保护
		}
	}
}
