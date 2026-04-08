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
