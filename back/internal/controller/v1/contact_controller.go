package v1

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
