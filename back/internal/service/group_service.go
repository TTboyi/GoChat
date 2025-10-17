package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// generateShortID 生成 8 位随机数字字符串
func generateGroupID() string {
	for {
		id := fmt.Sprintf("%06d", rand.Intn(1000000))
		var count int64
		db := config.GetDB()
		db.Model(&model.GroupInfo{}).Where("uuid = ?", id).Count(&count)
		if count == 0 {
			return id
		}
	} // 0~99999999
}

func CreateGroup(req *req.CreateGroupRequest) (string, error) {
	db := config.GetDB()

	// 构造群聊基本信息
	uuid6 := generateGroupID()
	group := model.GroupInfo{
		Uuid:      uuid6,
		Name:      req.Name,
		Notice:    req.Notice,
		OwnerId:   req.OwnerId,
		MemberCnt: 1,
		AddMode:   int8(req.AddMode),
		Avatar:    req.Avatar,
		Status:    0,
		CreatedAt: time.Now(),
	}

	// 成员初始化（仅群主）
	members := []string{req.OwnerId}
	membersJSON, err := json.Marshal(members)
	if err != nil {
		return "成员序列化失败", err
	}
	group.Members = membersJSON

	// 创建群聊
	if err := db.Create(&group).Error; err != nil {
		return "群聊创建失败", err
	}

	// 将群聊作为联系人添加给群主
	contact := model.UserContact{
		UserId:      req.OwnerId,
		ContactId:   group.Uuid,
		ContactType: 1, // 假设 1 表示群聊
		Status:      0,
		CreatedAt:   time.Now(),
	}

	if err := db.Create(&contact).Error; err != nil {
		return "联系人添加失败", err
	}

	return "创建成功", nil
}

func GetMyCreatedGroups(ownerId string) ([]model.GroupInfo, error) {
	db := config.GetDB()
	var groups []model.GroupInfo
	if err := db.Where("owner_id = ?", ownerId).Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func GetGroupAddMode(groupUuid string) (int8, error) {
	db := config.GetDB()
	var group model.GroupInfo
	if err := db.Select("add_mode").Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return 0, err
	}
	return group.AddMode, nil
}

func EnterGroup(userId, groupUuid, message string) error {
	db := config.GetDB()

	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return errors.New("群聊不存在")
	}

	// 判断加群方式
	if group.AddMode == 0 {
		// 直接加入
		var members []string
		_ = json.Unmarshal(group.Members, &members)
		for _, m := range members {
			if m == userId {
				return errors.New("已在群聊中")
			}
		}
		members = append(members, userId)

		membersBytes, _ := json.Marshal(members)
		group.Members = membersBytes
		group.MemberCnt = len(members)

		if err := db.Save(&group).Error; err != nil {
			return errors.New("加入群聊失败")
		}

		var cnt int64
		db.Model(&model.UserContact{}).
			Where("user_id = ? AND contact_id = ? AND contact_type = 1", userId, group.Uuid).
			Count(&cnt)
		if cnt == 0 {
			uc := model.UserContact{
				UserId:      userId,
				ContactId:   group.Uuid,
				ContactType: 1, // 群
				Status:      0,
				CreatedAt:   time.Now(),
			}
			if err := db.Create(&uc).Error; err != nil {
				return errors.New("加入群聊成功，但创建联系人失败")
			}
		}
		return nil
	}

	// 审核模式 → 插入一条申请记录
	apply := model.ContactApply{
		Uuid:        "A" + uuid.NewString()[:7],
		UserId:      userId,
		ContactId:   group.Uuid,
		ContactType: 1, // 群聊
		Status:      0, // 申请中
		Message:     message,
		LastApplyAt: time.Now(),
	}
	if err := db.Create(&apply).Error; err != nil {
		return errors.New("申请失败")
	}

	return nil
}

func LeaveGroup(userId, groupUuid string) error {
	db := config.GetDB()

	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return errors.New("群聊不存在")
	}

	if group.OwnerId == userId {
		return errors.New("群主不能直接退出群聊，请解散群聊")
	}

	var members []string
	_ = json.Unmarshal(group.Members, &members)

	newMembers := []string{}
	for _, m := range members {
		if m != userId {
			newMembers = append(newMembers, m)
		}
	}

	group.Members, _ = json.Marshal(newMembers)
	group.MemberCnt = len(newMembers)

	if err := db.Save(&group).Error; err != nil {
		return errors.New("退出群聊失败")
	}
	return nil
}

func GetGroupMemberList(groupUuid string) ([]string, error) {
	db := config.GetDB()

	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return nil, errors.New("群聊不存在")
	}

	var members []string
	if len(group.Members) > 0 {
		if err := json.Unmarshal(group.Members, &members); err != nil {
			return nil, errors.New("解析成员列表失败")
		}
	}

	return members, nil
}

// 移除群成员（群主操作）
func RemoveGroupMember(ownerId, groupUuid, targetUserId string) error {
	db := config.GetDB()

	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return errors.New("群聊不存在")
	}

	// 只允许群主踢人
	if group.OwnerId != ownerId {
		return errors.New("只有群主才能移除成员")
	}

	// 解析群成员
	var members []string
	if len(group.Members) > 0 {
		if err := json.Unmarshal(group.Members, &members); err != nil {
			return errors.New("解析成员失败")
		}
	}

	// 检查目标用户是否在群里
	found := false
	newMembers := make([]string, 0, len(members))
	for _, m := range members {
		if m == targetUserId {
			found = true
			continue // 跳过目标用户
		}
		newMembers = append(newMembers, m)
	}
	if !found {
		return errors.New("该用户不在群聊中")
	}

	// 更新群组信息
	group.Members, _ = json.Marshal(newMembers)
	group.MemberCnt = len(newMembers)

	if err := db.Save(&group).Error; err != nil {
		return errors.New("更新群聊失败")
	}

	return nil
}

// 解散群聊（群主操作）
func DismissGroup(ownerId, groupUuid string) error {
	db := config.GetDB()

	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return errors.New("群聊不存在")
	}

	// 只有群主能解散
	if group.OwnerId != ownerId {
		return errors.New("只有群主才能解散群聊")
	}

	// 更新状态为解散，清空成员
	group.Status = 2 // 2 = 解散
	group.Members = []byte("[]")
	group.MemberCnt = 0

	if err := db.Save(&group).Error; err != nil {
		return errors.New("解散失败")
	}

	return nil
}
func GetGroupInfo(groupUuid string) (*model.GroupInfo, error) {
	db := config.GetDB()
	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return nil, errors.New("群聊不存在")
	}
	return &group, nil
}

// 更新公告
func UpdateGroupNotice(userId, groupUuid, notice string) error {
	db := config.GetDB()
	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return errors.New("群聊不存在")
	}
	if group.OwnerId != userId {
		return errors.New("没有权限修改公告")
	}
	return db.Model(&group).Update("notice", notice).Error
}

// 更新群名
func UpdateGroupName(userId, groupUuid, newName string) error {
	db := config.GetDB()
	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return errors.New("群聊不存在")
	}
	if group.OwnerId != userId {
		return errors.New("没有权限修改群名称")
	}
	return db.Model(&group).Update("name", newName).Error
}
