// ============================================================
// 文件：back/internal/chat/server.go
// 作用：定义"在线用户路由中心"（ChatServer），管理所有 WebSocket 连接，
//       并负责消息分发、在线状态广播、群聊订阅等核心实时能力。
//
// 核心数据结构：
//   Server.Clients - map[string][]*Client
//     key:   用户的 UUID（字符串）
//     value: 该用户当前所有活跃的 WebSocket 连接切片
//     支持多端同时在线（手机+电脑同时登录）
//
//   groupMembers - map[string]map[string]bool
//     外层 key: 群聊的 UUID
//     内层 key: 已订阅该群的用户 UUID
//     内层 value: true（用 bool 作为 Set 的常见 Go 惯用写法）
//     这是内存中的"群聊在线订阅表"，不是真正的群成员表（那个在数据库里）
//
// 在线状态广播机制：
//   当用户 A 连接时：
//     1. 把 A 的连接加入 Clients["A的UUID"]
//     2. 如果是 A 的第一个连接，向所有其他在线用户广播 {"action":"user_online","userId":"A的UUID"}
//     3. 同时把当前所有在线用户列表发给 A（让 A 的界面立刻显示谁在线）
//   当用户 A 断开时：
//     1. 从 Clients["A的UUID"] 里删掉这条连接
//     2. 如果 A 所有连接都断了，广播 {"action":"user_offline","userId":"A的UUID"}
//     3. 清除 A 在所有群的订阅记录（groupMembers 里）
//
// 为什么消息主链路走 Kafka 而不是直接内存传递？
//   内存传递（旧方案）：Server → 直接找到目标用户连接 → 写消息
//     优点：简单快速
//     缺点：服务重启丢消息、多机部署时找不到连接
//   Kafka 方案（新主链路）：WebSocket → Kafka 队列 → Consumer → WebSocket
//     优点：消息不丢失（队列持久化）、天然支持多消费者（持久化、推送、缓存）
//     缺点：引入了额外的延迟（通常 <10ms，用户感知不到）
// ============================================================

package chat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"chatapp/back/internal/config"
	"chatapp/back/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ========= 供 client.go / ws.go 共用的消息结构 =========

// ChatEnvelope 是“进入后端消息主链路前”的统一包裹。
// WebSocket 层把前端 JSON 解析成它，之后无论是发 Kafka、分流到私聊/群聊，
// 还是写库，都尽量围绕这个结构展开。
type ChatEnvelope struct {
	Type      int8   `json:"type"`
	Content   string `json:"content,omitempty"`
	Url       string `json:"url,omitempty"`
	FileName  string `json:"fileName,omitempty"`
	FileType  string `json:"fileType,omitempty"`
	FileSize  string `json:"fileSize,omitempty"`
	SendId    string `json:"sendId"`
	ReceiveId string `json:"receiveId"`
	LocalId   string `json:"localId,omitempty"` // 前端生成，用于乐观更新
}

// OutgoingMessage 是发回前端的标准消息格式。
// 它和数据库模型 Message 很像，但职责不同：
// - Message 面向持久化；
// - OutgoingMessage 面向前端实时展示。
type OutgoingMessage struct {
	Uuid       string `json:"uuid"`
	LocalId    string `json:"localId,omitempty"`
	Type       int8   `json:"type"`
	Content    string `json:"content,omitempty"`
	Url        string `json:"url,omitempty"`
	FileName   string `json:"fileName,omitempty"`
	FileType   string `json:"fileType,omitempty"`
	FileSize   string `json:"fileSize,omitempty"`
	SendId     string `json:"sendId"`
	SendName   string `json:"sendName"`
	SendAvatar string `json:"sendAvatar"`
	ReceiveId  string `json:"receiveId"`
	CreatedAt  int64  `json:"createdAt"`
}

// CallSignal 用于 WebRTC 信令转发。
// 真正的音视频流不经过后端；后端只负责在双方之间转发 offer/answer/candidate 等控制消息。
type CallSignal struct {
	Action    string `json:"action"`   // call_invite / call_answer / call_candidate / call_end
	CallId    string `json:"callId"`   // 通话唯一ID
	From      string `json:"from"`     // 主叫
	To        string `json:"to"`       // 被叫
	CallType  string `json:"callType"` // audio / video
	Accept    *bool  `json:"accept,omitempty"`
	Content   string `json:"content,omitempty"` // SDP / ICE
	CreatedAt int64  `json:"createdAt"`
}

// ======================================================

// Server 是内存态在线路由中心。
// 即使当前消息主干已迁移到 Kafka，它仍然承担“在线连接管理、信令分发、群订阅状态”这些职责。
// 改造后的 Clients 结构支持“同一 userId 多端同时在线”。
type Server struct {
	Clients  map[string][]*Client // 在线用户：userUuid -> []*Client（支持多端）
	Mutex    *sync.Mutex
	Transmit chan ChatEnvelope // 旧版内存消息入口（当前主要作为演进痕迹保留）
	Login    chan *Client
	Logout   chan *Client
}

// groupMembers 记录“当前哪些在线用户已经订阅了某个群”。
// 这并不是群的真实成员表；真实成员仍以数据库为准。
var (
	groupMembers = make(map[string]map[string]bool) // groupId -> userId -> 在线状态
	groupMemsMu  sync.RWMutex
)

// ChatServer 是全局唯一的在线路由中心。
var ChatServer = &Server{
	Clients:  make(map[string][]*Client),
	Mutex:    &sync.Mutex{},
	Transmit: make(chan ChatEnvelope, 1024),
	Login:    make(chan *Client, 128),
	Logout:   make(chan *Client, 128),
}

// AddUserToGroup 在内存中登记某个在线用户已订阅某个群。
// 前端建立 WS 后会主动发送 join_group，让后端知道“这个连接愿意接收哪个群的推送”。
func (s *Server) AddUserToGroup(userId, groupId string) {
	groupMemsMu.Lock()
	defer groupMemsMu.Unlock()
	if _, ok := groupMembers[groupId]; !ok {
		groupMembers[groupId] = make(map[string]bool)
	}
	groupMembers[groupId][userId] = true
	slog.Info("join_group", "user_id", userId, "group_id", groupId)
}

// Run 启动消息循环
func (s *Server) Run() {
	for {
		select {
		// 新客户端登录
		case client := <-s.Login:
			s.Mutex.Lock()
			s.Clients[client.Uuid] = append(s.Clients[client.Uuid], client)
			s.Mutex.Unlock()

			slog.Info("ws_login", "user_id", client.Uuid)
			// 给用户一个欢迎消息
			msg := fmt.Sprintf("欢迎用户 %s 加入聊天室", client.Uuid)
			client.SendBack <- []byte(msg)

		// 客户端退出（WebSocket 断开）
		case client := <-s.Logout:
			s.RemoveClient(client)
			slog.Info("ws_logout", "user_id", client.Uuid)

		// 收到消息
		case env := <-s.Transmit:
			// 统一走路由 + 入库 + 下发（点对点/群聊都在这里处理）
			if err := s.routeAndPersist(env); err != nil {
				slog.Error("route_and_persist_failed", "err", err)
			}

		}

	}
}

// ForwardCallSignal 把通话控制消息转发给目标用户。
// 这是“信令服务器”角色：只转发建立连接所需的控制数据，不承载媒体流本身。
func (s *Server) ForwardCallSignal(from string, req ChatMessageRequest) {
	sig := CallSignal{
		Action:    req.Action,
		CallId:    req.CallId,
		From:      from,
		To:        req.ReceiveId,
		CallType:  req.CallType,
		Accept:    req.Accept,
		Content:   req.Content,
		CreatedAt: time.Now().Unix(),
	}

	raw, _ := json.Marshal(sig)
	slog.Info("call_signal_forward", "from", from, "to", req.ReceiveId, "action", req.Action)

	// 发给对方
	s.DeliverToUser(req.ReceiveId, raw)
	// 也回传给自己（确认状态）
	//s.DeliverToUser(from, raw)
}

// ============== 核心：路由 & 存库 ==============

// routeAndPersist 根据 receiveId 判断消息是发给个人还是群，
// 然后进入不同的持久化/分发逻辑。
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

// isGroupTarget 判断 receiveId 是否表示一个群。
// 这里采用“两级判断”：
// 1. 先查内存订阅表，速度快；
// 2. 若内存里没有，再用数据库兜底。
func (s *Server) isGroupTarget(db *gorm.DB, id string) bool {
	if id == "" {
		return false
	}
	// ① 快速判断：有订阅（有人 join 过）就认为是群
	groupMemsMu.RLock()
	_, ok := groupMembers[id]
	groupMemsMu.RUnlock()
	if ok {
		return true
	}
	// ② 数据库兜底判断
	var cnt int64
	if err := db.Model(&model.GroupInfo{}).Where("uuid = ?", id).Count(&cnt).Error; err == nil && cnt > 0 {
		return true
	}
	return false
}

// handleDirect 处理点对点消息。
// 它做三件事：确保会话存在、写入数据库、把消息实时推给发送方和接收方的在线端。
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
	displayName := nz(senderName, "用户")
	out := OutgoingMessage{
		Uuid:       msgID,
		Type:       env.Type,
		Content:    env.Content,
		Url:        env.Url,
		FileName:   env.FileName,
		FileType:   env.FileType,
		FileSize:   env.FileSize,
		SendId:     env.SendId,
		SendName:   displayName,
		SendAvatar: senderAvatar,
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
		SendName:   displayName,
		SendAvatar: senderAvatar,
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

	// 3) 下发：对方所有端 & 自己所有端（回显）
	raw, _ := json.Marshal(out)
	s.deliverToUser(env.ReceiveId, raw) // 对方在线才会收到
	s.deliverToUser(env.SendId, raw)    // 自己所有端回显（多端同步）
	// 成功下发至少一份即可视为"已发送"
	_ = db.Model(&model.Message{}).Where("uuid = ?", msgID).Update("status", 1).Error

	_ = recvName
	_ = recvAvatar
	return nil
}

// handleGroup 处理群聊消息。
// 和私聊相比，最大的区别是它会广播给当前已订阅该群的所有在线用户。
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
	displayNameG := nz(senderName, "用户")
	out := OutgoingMessage{
		Uuid:       msgID,
		Type:       env.Type,
		Content:    env.Content,
		Url:        env.Url,
		FileName:   env.FileName,
		FileType:   env.FileType,
		FileSize:   env.FileSize,
		SendId:     env.SendId,
		SendName:   displayNameG,
		SendAvatar: senderAvatar,
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
		SendName:   displayNameG,
		SendAvatar: senderAvatar,
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
	groupMemsMu.RLock()
	subs := make(map[string]bool, len(groupMembers[env.ReceiveId]))
	for k, v := range groupMembers[env.ReceiveId] {
		subs[k] = v
	}
	groupMemsMu.RUnlock()
	for userId := range subs {
		s.deliverToUser(userId, raw)
	}

	_ = db.Model(&model.Message{}).Where("uuid = ?", msgID).Update("status", 1).Error
	return nil
}

// deliverToUser 将消息投递给某个用户的所有在线连接，实现多端同步。
func (s *Server) deliverToUser(userId string, raw []byte) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	conns, ok := s.Clients[userId]
	if !ok || len(conns) == 0 {
		return
	}
	slog.Debug("deliver_to_user", "user_id", userId, "conn_count", len(conns))
	for _, c := range conns {
		select {
		case c.SendBack <- raw:
		default:
			slog.Warn("deliver_dropped", "user_id", userId)
		}
	}
}

// ✅ 推送"群已解散"通知
func (s *Server) PushGroupDismiss(groupId string) {
	raw, _ := json.Marshal(map[string]interface{}{
		"uuid":      newIDWithPrefix("SYS"),
		"type":      99, // ✅ 自定义类型：99 表示系统事件
		"receiveId": groupId,
		"content":   "group_dismiss", // ✅ 事件内容
		"createdAt": time.Now().Unix(),
	})

	// ✅ 推给这个群的所有在线成员
	groupMemsMu.RLock()
	subs := make(map[string]bool, len(groupMembers[groupId]))
	for k, v := range groupMembers[groupId] {
		subs[k] = v
	}
	groupMemsMu.RUnlock()
	for uid := range subs {
		s.deliverToUser(uid, raw)
	}
}

// ============== 会话兜底 & 基础查询 ==============

// ensureSessionForDirect 确保私聊会话存在。
// 这里的设计说明：消息并不会隐式“只存在于 message 表”；
// session 表单独维护聊天列表项，让会话列表查询更直接。
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

// ensureSessionForGroup 确保“某用户看见的某个群聊会话”存在。
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

// loadUserBasic 读取消息渲染所需的最小用户资料。
func loadUserBasic(db *gorm.DB, userId string) (nickname, avatar string, err error) {
	var u model.UserInfo
	if e := db.Where("uuid = ?", userId).First(&u).Error; e != nil {
		return "", "", e
	}
	return u.Nickname, u.Avatar, nil
}

// ============== 工具 ==============

// newIDWithPrefix 生成一个带业务前缀的短 ID。
// M/S/SYS 这类前缀可以帮助阅读数据库记录时快速判断对象类型。
func newIDWithPrefix(p string) string {
	// 生成 19 位随机（去掉 - 的 uuid），再加 1 位前缀，正好 20
	raw := strings.ReplaceAll(uuid.New().String(), "-", "")
	return p + raw[:19]
}

// nz 是一个“小而频繁”的兜底工具：当字符串为空时返回默认值。
func nz(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// AddClient 把一个新的 WebSocket 连接挂到在线表里。
// 除了保存连接外，它还会主动同步在线用户列表，并在“某用户首次上线”时广播在线事件。
func (s *Server) AddClient(c *Client) {
	s.Mutex.Lock()

	// 收集当前在线用户ID（去重）
	existingIds := make([]string, 0)
	for uid := range s.Clients {
		if uid != c.Uuid {
			existingIds = append(existingIds, uid)
		}
	}

	// 追加连接（不覆盖）
	s.Clients[c.Uuid] = append(s.Clients[c.Uuid], c)
	slog.Info("ws_connect", "user_id", c.Uuid, "total_conns", totalConnections(s.Clients), "user_conns", len(s.Clients[c.Uuid]))

	// 发送在线用户列表给新客户端（仅包括其他用户，不含自己）
	onlineMsg, _ := json.Marshal(map[string]interface{}{
		"action":  "online_users",
		"userIds": existingIds,
	})
	select {
	case c.SendBack <- onlineMsg:
	default:
	}

	// 如果是该用户的第一个连接，才广播 user_online 给其他用户
	if len(s.Clients[c.Uuid]) == 1 {
		onlineNotify, _ := json.Marshal(map[string]interface{}{
			"action": "user_online",
			"userId": c.Uuid,
		})
		for uid, clients := range s.Clients {
			if uid == c.Uuid {
				continue
			}
			for _, client := range clients {
				select {
				case client.SendBack <- onlineNotify:
				default:
				}
			}
		}
	}

	s.Mutex.Unlock()
}

// totalConnections 统计所有连接总数（调试用）
func totalConnections(m map[string][]*Client) int {
	total := 0
	for _, conns := range m {
		total += len(conns)
	}
	return total
}

func keysOfClients(m map[string][]*Client) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// removeUserFromAllGroups 清理用户在内存订阅表中的所有群订阅。
// 当用户最后一个连接断开时，不应继续向该用户推送群消息。
func (s *Server) removeUserFromAllGroups(userId string) {
	groupMemsMu.Lock()
	defer groupMemsMu.Unlock()
	for gid, subs := range groupMembers {
		if subs[userId] {
			delete(subs, userId)
			if len(subs) == 0 {
				delete(groupMembers, gid)
			}
		}
	}
}

// RemoveClient 精确移除某一个连接。
// 如果这已经是该用户最后一个连接，还会触发离线广播。
func (s *Server) RemoveClient(c *Client) {
	s.Mutex.Lock()

	userId := c.Uuid
	conns := s.Clients[userId]
	newConns := make([]*Client, 0, len(conns))
	for _, conn := range conns {
		if conn != c {
			newConns = append(newConns, conn)
		}
	}

	if len(newConns) == 0 {
		delete(s.Clients, userId)
		slog.Info("ws_all_disconnected", "user_id", userId)
		s.removeUserFromAllGroups(userId)

		// 广播 user_offline（只有最后一个连接断开时才广播）
		offlineNotify, _ := json.Marshal(map[string]interface{}{
			"action": "user_offline",
			"userId": userId,
		})
		for _, clients := range s.Clients {
			for _, client := range clients {
				select {
				case client.SendBack <- offlineNotify:
				default:
				}
			}
		}
	} else {
		s.Clients[userId] = newConns
		slog.Info("ws_disconnect_partial", "user_id", userId, "remaining_conns", len(newConns))
	}

	_ = c.Conn.Close()
	s.Mutex.Unlock()
}

// RemoveAllClients 强制移除某个用户的所有连接，常见于主动登出或管理员踢下线。
func (s *Server) RemoveAllClients(userId string) {
	s.Mutex.Lock()
	conns, ok := s.Clients[userId]
	if ok {
		for _, c := range conns {
			_ = c.Conn.Close()
		}
		delete(s.Clients, userId)
		s.removeUserFromAllGroups(userId)
		slog.Info("ws_force_removed", "user_id", userId)
	}

	// 广播离线
	offlineNotify, _ := json.Marshal(map[string]interface{}{
		"action": "user_offline",
		"userId": userId,
	})
	for _, clients := range s.Clients {
		for _, client := range clients {
			select {
			case client.SendBack <- offlineNotify:
			default:
			}
		}
	}
	s.Mutex.Unlock()
}

// DeliverToUser 是对外暴露的推送入口，供其它包发送控制消息或系统通知。
func (s *Server) DeliverToUser(userId string, raw []byte) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	conns, ok := s.Clients[userId]
	if !ok || len(conns) == 0 {
		slog.Debug("deliver_user_offline", "user_id", userId)
		return
	}
	slog.Debug("deliver_to_user", "user_id", userId, "conn_count", len(conns))

	for _, c := range conns {
		select {
		case c.SendBack <- raw:
		default:
			// 下行拥塞保护
		}
	}
}
