// ============================================================
// 文件：back/internal/chat/session_id.go
// 作用：根据消息的收发方信息生成"会话ID"，用作 Redis 缓存的 Key。
//
// buildSessionId 的设计原理：
//   群聊：直接用 "G:" 前缀 + groupUuid，简单明了
//   私聊：把 sendId 和 receiveId 排序后拼接
//         为什么要排序？
//         用户 A（uuid="abc"）给用户 B（uuid="xyz"）发消息：sendId="abc", recvId="xyz" → "abc:xyz"
//         用户 B（uuid="xyz"）给用户 A（uuid="abc"）发消息：sendId="xyz", recvId="abc" → "abc:xyz"（排序后一样）
//         这样 A→B 和 B→A 的消息都存在同一个缓存桶里，查询时不用查两遍
// ============================================================

package chat

func buildSessionId(km *KafkaMessage) string {
	if isGroup(km.ReceiveId) {
		return "G:" + km.ReceiveId
	}

	// 单聊：小的在前，保证一致
	if km.SendId < km.ReceiveId {
		return km.SendId + ":" + km.ReceiveId
	}
	return km.ReceiveId + ":" + km.SendId
}
