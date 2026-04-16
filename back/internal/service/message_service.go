// ============================================================
// 文件：back/internal/service/message_service.go
// 作用：消息相关的业务逻辑：历史消息查询、消息撤回、标记已读、清空会话。
//
// GetMessageList（获取私聊历史）：
//   先查数据库（不用 Redis 缓存）的理由：
//   - 历史消息分页需要精确的 offset/limit 语义（Redis LIST 不支持）
//   - 撤回状态、已读状态需要实时准确，缓存可能有延迟
//   - 分页使用 beforeTime（时间戳游标分页，比 OFFSET 分页更稳定）：
//     每次加载"比上次最旧一条消息还早的消息"，滚动加载历史不会因为新消息插入而错位
//
// RecallMessage（撤回消息）：
//   两个业务规则的实现：
//   1. sendId 必须等于当前登录用户（只能撤回自己发的）
//   2. 发送时间到现在不能超过 10 分钟（超时不可撤回，与微信等 APP 一致）
//   撤回不是真正删除，而是把 is_recalled = 1、content 清空，
//   前端看到 is_recalled = 1 就显示"此消息已撤回"。
//
// MarkMessagesRead（标记已读）：
//   批量更新：WHERE receive_id=我 AND send_id=对方 AND read_at IS NULL
//   用"批量更新"而不是"逐条更新"，减少数据库操作次数。
//   打开某个会话窗口时调用，把这个会话里所有未读消息标为已读。
// ============================================================
package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/model"
	"context"
	"errors"
	"time"
)

// GetMessageList 获取两个人之间的历史消息。
// 它按 created_at 倒序查数据库，再由前端在展示前 reverse，
// 这样“加载更多历史消息”时更容易基于最老一条消息的时间戳做分页。
func GetMessageList(userId, targetId string, limit int, beforeTime int64) ([]model.Message, error) {
	// 这里直接查 DB，而不是先查 Redis，
	// 因为历史消息列表通常需要准确的分页、撤回状态和已读状态。
	db := config.GetDB()
	var list []model.Message

	if limit <= 0 {
		limit = 50
	}

	q := db.Where(
		"(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)",
		userId, targetId, targetId, userId,
	)
	if beforeTime > 0 {
		q = q.Where("UNIX_TIMESTAMP(created_at) < ?", beforeTime)
	}
	err := q.Order("created_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

// GetGroupMessageList 获取群聊历史消息，分页逻辑与私聊保持一致。
// 调用前会校验 userId 是否是该群成员，防止越权读取不属于自己的群消息。
func GetGroupMessageList(userId, groupId string, limit int, beforeTime int64) ([]model.Message, error) {
	if !IsGroupMember(userId, groupId) {
		return nil, errors.New("无权限访问该群消息")
	}

	db := config.GetDB()
	var list []model.Message

	if limit <= 0 {
		limit = 50
	}

	q := db.Where("receive_id = ?", groupId)
	if beforeTime > 0 {
		q = q.Where("UNIX_TIMESTAMP(created_at) < ?", beforeTime)
	}
	err := q.Order("created_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

// SaveMessage 作为最薄的一层持久化包装，主要为了给其它调用方提供统一入口。
func SaveMessage(msg *model.Message) error {
	db := config.GetDB()
	return db.Create(msg).Error
}

// RecallMessage 实现“发送后 10 分钟内可撤回”的业务规则。
func RecallMessage(senderId, msgId string) error {
	db := config.GetDB()
	var msg model.Message
	if err := db.Where("uuid = ?", msgId).First(&msg).Error; err != nil {
		return errors.New("消息不存在")
	}
	if msg.SendId != senderId {
		return errors.New("只能撤回自己的消息")
	}
	if time.Since(msg.CreatedAt) > 10*time.Minute {
		return errors.New("超过10分钟，无法撤回")
	}
	return db.Model(&msg).Updates(map[string]interface{}{
		"is_recalled": 1,
		"content":     "",
	}).Error
}

// MarkMessagesRead 把某个会话里“对方发给我、我还没读”的消息统一标为已读。
func MarkMessagesRead(receiverId, senderId string) error {
	db := config.GetDB()
	now := time.Now()
	return db.Model(&model.Message{}).
		Where("receive_id = ? AND send_id = ? AND read_at IS NULL AND is_recalled = 0", receiverId, senderId).
		Update("read_at", now).Error
}

// GetUnreadCount 统计某个会话的未读消息数，常用于会话列表角标。
func GetUnreadCount(ctx context.Context, receiverId, senderId string) (int64, error) {
	db := config.GetDB()
	var count int64
	err := db.Model(&model.Message{}).
		Where("receive_id = ? AND send_id = ? AND read_at IS NULL AND is_recalled = 0", receiverId, senderId).
		Count(&count).Error
	return count, err
}

// ClearConversation 在删除好友时把双方私聊消息一并清除。
// 这里做的是物理删除，所以之后历史记录无法恢复。
func ClearConversation(userId, targetId string) error {
	db := config.GetDB()
	return db.
		Where(
			"(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)",
			userId, targetId, targetId, userId,
		).
		Delete(&model.Message{}).Error
}
