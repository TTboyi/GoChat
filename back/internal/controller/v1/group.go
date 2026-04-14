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

// / EnterGroupDirectly 用户申请加入群聊（当前逻辑：直接加入）
func EnterGroupDirectly(c *gin.Context) {
	userId := c.GetString("userId")
	groupUuid := c.PostForm("groupId")
	message := c.PostForm("message")

	if groupUuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 groupId"})
		return
	}
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	db := config.GetDB()

	// 1) 读群，拿 “加入之前”的成员，作为广播对象（老成员）
	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "群不存在"})
		return
	}
	var oldMembers []string
	_ = json.Unmarshal(group.Members, &oldMembers)

	// 2) 执行加入（你原本已有的业务）
	if err := service.EnterGroup(userId, groupUuid, message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3) 异步通知老成员：有人加入（不推给新加入者）
	go func(gid, joinUid string, receivers []string) {
		payload := map[string]any{
			"action":  "group_join",
			"groupId": gid,
			"userId":  joinUid,
		}
		raw, _ := json.Marshal(payload)
		for _, uid := range receivers {
			if uid != joinUid {
				chat.ChatServer.DeliverToUser(uid, raw)
			}
		}
	}(groupUuid, userId, oldMembers)

	// 4) （可选）通知新加入者：自己入群成功，用来做 UX 提示，不需要的话可以删掉
	// go func(gid, joinUid string) {
	//     payload := map[string]any{
	//         "action":  "group_join_ack",
	//         "groupId": gid,
	//     }
	//     raw, _ := json.Marshal(payload)
	//     chat.ChatServer.DeliverToUser(joinUid, raw)
	// }(groupUuid, userId)

	c.JSON(http.StatusOK, gin.H{"message": "申请已提交"})
}

// ✅ 退出群聊（成员自己退出）
func QuitGroup(c *gin.Context) {
	var req struct {
		GroupId string `json:"groupId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 从 JWT 中间件获取当前用户 ID，防止伪造
	userId := c.GetString("userId")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	db := config.GetDB()
	var group model.GroupInfo
	if err := db.First(&group, "uuid = ?", req.GroupId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "群不存在"})
		return
	}

	// 解 JSON，拿退出前的成员列表用于广播
	var members []string
	_ = json.Unmarshal(group.Members, &members)

	// 使用 service 层（负责更新 members 并清理 user_contact）
	if err := service.LeaveGroup(userId, req.GroupId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ✅ 广播给其它成员：有人退群（仅通知剩余成员，不重复发送）
	go func() {
		msg := map[string]any{
			"action":  "group_quit",
			"groupId": req.GroupId,
			"userId":  userId,
		}
		raw, _ := json.Marshal(msg)
		for _, uid := range members {
			if uid != userId {
				chat.ChatServer.DeliverToUser(uid, raw)
			}
		}
		// 通知退出者本身（客户端可据此做 UI 清理）
		chat.ChatServer.DeliverToUser(userId, raw)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "已退出群聊"})
}

// 查询群聊成员列表
func GetGroupMemberList(c *gin.Context) {
	userId := c.GetString("userId")
	groupUuid := c.Query("groupUuid")
	if groupUuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少群组UUID"})
		return
	}

	if !service.IsGroupMember(userId, groupUuid) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限查看该群成员"})
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
	userId := c.GetString("userId")
	groupId := c.Query("groupId")
	db := config.GetDB()

	var group model.GroupInfo
	if err := db.First(&group, "uuid = ?", groupId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "群不存在"})
		return
	}

	// 反序列化成员列表，同时做鉴权
	var members []string
	_ = json.Unmarshal(group.Members, &members)

	isMember := false
	for _, m := range members {
		if m == userId {
			isMember = true
			break
		}
	}
	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限查看该群详情"})
		return
	}

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

// UpdateGroupAvatar 更新群头像（群主权限）
func UpdateGroupAvatar(c *gin.Context) {
	userId := c.GetString("userId")
	var form struct {
		GroupUuid string `json:"groupUuid" form:"groupUuid" binding:"required"`
		Avatar    string `json:"avatar" form:"avatar" binding:"required"`
	}
	if err := c.ShouldBind(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数"})
		return
	}
	if err := service.UpdateGroupAvatar(userId, form.GroupUuid, form.Avatar); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "群头像已更新"})
}
