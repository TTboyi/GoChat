package v1

import (
	"encoding/json"
	"net/http"

	"chatapp/back/internal/chat"
	"chatapp/back/internal/config"
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/model"
	"chatapp/back/internal/service"

	// ✅ 记得加这个

	"github.com/gin-gonic/gin"
)

// 既兼容 JSON 也兼容表单
type dismissReq struct {
	GroupId string `json:"groupId" form:"groupId" binding:"required"`
}

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

// ✅ 退出群聊（成员自己退出）
func QuitGroup(c *gin.Context) {
	var req struct {
		GroupId string `json:"groupId" binding:"required"`
		UserId  string `json:"userId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	db := config.GetDB()
	var group model.GroupInfo
	if err := db.First(&group, "uuid = ?", req.GroupId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "群不存在"})
		return
	}

	// 解 JSON
	var members []string
	_ = json.Unmarshal(group.Members, &members)

	// 删除该成员
	newMembers := make([]string, 0, len(members))
	for _, id := range members {
		if id != req.UserId {
			newMembers = append(newMembers, id)
		}
	}

	// 保存
	membersJSON, _ := json.Marshal(newMembers)
	db.Model(&group).Updates(map[string]interface{}{
		"members":    membersJSON,
		"member_cnt": len(newMembers),
	})

	// ✅ 广播给其它成员：有人退群
	go func() {
		msg := map[string]any{
			"action":  "group_quit",
			"groupId": req.GroupId,
			"userId":  req.UserId,
		}
		raw, _ := json.Marshal(msg)
		for _, uid := range newMembers {
			chat.ChatServer.DeliverToUser(uid, raw)
		}

		chat.ChatServer.DeliverToUser(req.UserId, raw)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "已退出群聊"})
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
func DismissGroupHandler(c *gin.Context) {
	var req dismissReq
	if err := c.ShouldBind(&req); err != nil || req.GroupId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 groupId"})
		return
	}

	// ✅ 从中间件读取用户ID（统一用 userId；兼容老的 ownerId）
	uid := c.GetString("userId")
	if uid == "" {
		uid = c.GetString("ownerId")
	}
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	// ✅ service 返回成员列表（在清空前取出）
	members, err := service.DismissGroup(uid, req.GroupId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ✅ 给所有成员推送 “group_dismissed” 控制消息
	go func(groupId string, uids []string) {
		payload := map[string]any{
			"action":   "group_dismissed",
			"groupId":  groupId,
			"operator": uid,
		}
		raw, _ := json.Marshal(payload)
		for _, m := range uids {
			chat.ChatServer.DeliverToUser(m, raw)
		}
	}(req.GroupId, members)

	c.JSON(http.StatusOK, gin.H{"message": "群聊已解散"})
}

// 获取群详情
func GetGroupInfo(c *gin.Context) {
	groupId := c.Query("groupId")
	db := config.GetDB()

	var group model.GroupInfo
	if err := db.First(&group, "uuid = ?", groupId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "群不存在"})
		return
	}

	// 反序列化成员
	var members []string
	_ = json.Unmarshal(group.Members, &members)

	c.JSON(http.StatusOK, gin.H{
		"uuid":        group.Uuid,
		"name":        group.Name,
		"notice":      group.Notice,
		"ownerId":     group.OwnerId,
		"memberCount": group.MemberCnt,
		"avatar":      group.Avatar,
		"members":     members,
	})
}

// 更新群公告
func UpdateGroupNotice(c *gin.Context) {
	userId := c.GetString("userId")
	groupUuid := c.PostForm("groupUuid")
	notice := c.PostForm("notice")

	if groupUuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数"})
		return
	}
	if err := service.UpdateGroupNotice(userId, groupUuid, notice); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "群公告已更新"})
}

// 更新群名称
func UpdateGroupName(c *gin.Context) {
	userId := c.GetString("userId")
	groupUuid := c.PostForm("groupUuid")
	newName := c.PostForm("name")

	if groupUuid == "" || newName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数"})
		return
	}
	if err := service.UpdateGroupName(userId, groupUuid, newName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "群名称已更新"})
}
