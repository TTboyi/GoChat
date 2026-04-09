package v1

import (
	"chatapp/back/internal/chat"
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/service"
	"encoding/json"
	"net/http"

	"fmt"

	"github.com/gin-gonic/gin"
)

// 申请添加联系人
func ApplyContact(c *gin.Context) {
	userId := c.GetString("userId")
	var form req.ApplyContactRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	targetUserId, err := service.ApplyContactByTarget(userId, form.Target, form.Message)
	if err != nil {
		fmt.Println("❌ 添加好友失败：", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ✅ 通过 WebSocket 通知对方有新的好友申请
	if targetUserId != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"action": "new_contact_apply",
			"from":   userId,
		})
		chat.ChatServer.DeliverToUser(targetUserId, payload)
	}

	c.JSON(http.StatusOK, gin.H{"message": "申请成功"})
}

// 获取新的联系人申请
func GetNewContactList(c *gin.Context) {
	userId := c.GetString("userId")
	list, err := service.GetNewContactApplyList(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// 审核联系人申请
func HandleContactApply(c *gin.Context) {
	userId := c.GetString("userId")
	var form req.HandleContactApplyRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	applicantId, err := service.HandleContactApply(userId, &form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ✅ 如果是同意申请，通知申请方刷新联系人列表
	if form.Approve && applicantId != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"action":    "contact_apply_accepted",
			"acceptorId": userId,
		})
		chat.ChatServer.DeliverToUser(applicantId, payload)
	}

	c.JSON(http.StatusOK, gin.H{"message": "处理成功"})
}

// 获取联系人列表
func GetContactList(c *gin.Context) {
	userId := c.GetString("userId")
	list, err := service.GetContactList(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// 删除联系人
func DeleteContact(c *gin.Context) {
	userId := c.GetString("userId")
	var form req.DeleteContactRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := service.DeleteContact(userId, form.TargetUserId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// 拉黑联系人
func BlackContact(c *gin.Context) {
	userId := c.GetString("userId")
	var form req.BlackContactRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := service.BlackContact(userId, form.TargetUserId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "拉黑失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已拉黑"})
}

// 解除拉黑联系人
func UnBlackContact(c *gin.Context) {
	userId := c.GetString("userId")
	var form req.UnBlackContactRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := service.UnBlackContact(userId, form.TargetUserId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已解除拉黑"})
}

// 拒绝联系人申请
func RefuseContactApply(c *gin.Context) {
	userId := c.GetString("userId")
	var form req.RefuseContactApplyRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := service.RefuseContactApply(userId, form.ApplyUuid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已拒绝"})
}

// 拉黑申请
func BlackApply(c *gin.Context) {
	userId := c.GetString("userId")
	var form req.BlackApplyRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := service.BlackApply(userId, form.ApplyUuid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已拉黑申请"})
}

// 获取我加入的群聊
func LoadMyJoinedGroup(c *gin.Context) {
	userId := c.GetString("userId")

	groups, err := service.GetMyJoinedGroups(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if len(groups) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": groups})
}

// 获取联系人信息
func GetContactInfo(c *gin.Context) {
	var form req.GetContactInfoRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	info, err := service.GetContactInfo(form.TargetId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": info})
}
