package v1

import (
	"net/http"

	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/service"

	"github.com/gin-gonic/gin"
)

func CreateGroup(c *gin.Context) {
	var req req.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数绑定失败: " + err.Error()})
		return
	}

	groupUUID, err := service.CreateGroup(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "群聊创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "群聊创建成功",
		"group_uuid": groupUUID,
	})
}

func LoadMyGroup(c *gin.Context) {
	userId := c.GetString("userId") // JWT 中间件注入
	groups, err := service.GetMyCreatedGroups(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "查询成功",
		"groups":  groups,
	})
}

func CheckGroupAddMode(c *gin.Context) {
	type req struct {
		Uuid string `json:"uuid" binding:"required"`
	}
	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	mode, err := service.GetGroupAddMode(r.Uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uuid":     r.Uuid,
		"add_mode": mode,
	})
}

// EnterGroupDirectly 用户申请加入群聊
func EnterGroupDirectly(c *gin.Context) {
	userId := c.GetString("userId")
	groupUuid := c.PostForm("groupId")
	message := c.PostForm("message")

	err := service.EnterGroup(userId, groupUuid, message)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "申请已提交"})
}

func LeaveGroup(c *gin.Context) {
	userId := c.GetString("userId")
	groupUuid := c.PostForm("groupUuid")

	err := service.LeaveGroup(userId, groupUuid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "退出成功"})
}

// 查询群聊成员列表
func GetGroupMemberList(c *gin.Context) {
	groupUuid := c.Query("groupUuid")
	if groupUuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少群组UUID"})
		return
	}

	members, err := service.GetGroupMemberList(groupUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groupUuid": groupUuid,
		"members":   members,
	})
}

// 移除群成员（群主权限）
func RemoveGroupMember(c *gin.Context) {
	ownerId := c.GetString("userId") // 当前登录用户
	groupUuid := c.PostForm("groupUuid")
	targetUserId := c.PostForm("targetUserId")

	if groupUuid == "" || targetUserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数"})
		return
	}

	if err := service.RemoveGroupMember(ownerId, groupUuid, targetUserId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "移除成功"})
}

// 解散群聊（群主权限）
func DismissGroup(c *gin.Context) {
	ownerId := c.GetString("userId") // 当前登录用户
	groupUuid := c.PostForm("groupUuid")

	if groupUuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少群组UUID"})
		return
	}

	if err := service.DismissGroup(ownerId, groupUuid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "群聊已解散"})
}
