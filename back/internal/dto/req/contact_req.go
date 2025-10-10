package req

// 申请添加联系人
type ApplyContactRequest struct {
	Target  string `json:"target" binding:"required"` // 被申请人 UUID
	Message string `json:"message"`                   // 附加消息
}

// 审核联系人申请
type HandleContactApplyRequest struct {
	ApplyUuid string `json:"applyUuid" binding:"required"`
	Approve   bool   `json:"approve"` // true 通过, false 拒绝
}

// 删除联系人
type DeleteContactRequest struct {
	TargetUserId string `json:"targetUserId" binding:"required"`
}

// 拉黑联系人
type BlackContactRequest struct {
	TargetUserId string `json:"targetUserId" binding:"required"`
}

// 解除拉黑联系人
type UnBlackContactRequest struct {
	TargetUserId string `json:"targetUserId" binding:"required"`
}

// 拒绝联系人申请
type RefuseContactApplyRequest struct {
	ApplyUuid string `json:"applyUuid" binding:"required"`
}

// 拉黑联系人申请
type BlackApplyRequest struct {
	ApplyUuid string `json:"applyUuid" binding:"required"`
}

// 获取联系人信息
type GetContactInfoRequest struct {
	TargetId string `json:"targetId" binding:"required"` // 可能是用户UUID或群UUID
}
