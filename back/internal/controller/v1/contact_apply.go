package v1

import (
	"chatapp/back/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 新建申请（一般在群聊 addMode=1 时调用）
func CreateGroupApply(c *gin.Context) {
	userId := c.GetString("userId")
	groupId := c.PostForm("groupUuid")
	message := c.PostForm("message")

	if err := service.CreateGroupApply(userId, groupId, message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "申请失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "申请成功"})
}

// 获取群聊待审核申请
func GetGroupApplyList(c *gin.Context) {
	groupId := c.Query("groupUuid")

	list, err := service.GetGroupApplyList(groupId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// 处理群聊申请
func HandleGroupApply(c *gin.Context) {
	applyUuid := c.PostForm("applyUuid")
	approve := c.PostForm("approve") == "true"

	if err := service.HandleGroupApply(applyUuid, approve); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "处理成功"})
}
