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
	"gorm.io/gorm"
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
	members := []string{req.OwnerId}
	membersJSON, err := json.Marshal(members)
	if err != nil {
		return "", fmt.Errorf("成员序列化失败: %w", err)
	}

	group := model.GroupInfo{
		Uuid:      uuid6,
		Name:      req.Name,
		Notice:    req.Notice,
		OwnerId:   req.OwnerId,
		MemberCnt: 1,
		AddMode:   int8(req.AddMode),
		Avatar:    req.Avatar,
		Status:    0,
		Members:   membersJSON,
		CreatedAt: time.Now(),
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&group).Error; err != nil {
			return fmt.Errorf("群聊创建失败: %w", err)
		}
		contact := model.UserContact{
			UserId:      req.OwnerId,
			ContactId:   group.Uuid,
			ContactType: 1,
			Status:      0,
			CreatedAt:   time.Now(),
		}
		return tx.Create(&contact).Error
	})
	if err != nil {
		return "", err
	}

	return uuid6, nil
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

		return db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&group).Updates(map[string]interface{}{
				"members":    membersBytes,
				"member_cnt": len(members),
			}).Error; err != nil {
				return errors.New("加入群聊失败")
			}
			// 确保 user_contact 记录存在
			var cnt int64
			tx.Model(&model.UserContact{}).
				Where("user_id = ? AND contact_id = ? AND contact_type = 1", userId, group.Uuid).
				Count(&cnt)
			if cnt == 0 {
				uc := model.UserContact{
					UserId:      userId,
					ContactId:   group.Uuid,
					ContactType: 1,
					Status:      0,
					CreatedAt:   time.Now(),
				}
				return tx.Create(&uc).Error
			}
			return nil
		})
	}

	// ✅ 审核模式下暂时不推，保留原逻辑
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

	newMembersJSON, _ := json.Marshal(newMembers)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&group).Updates(map[string]interface{}{
			"members":    newMembersJSON,
			"member_cnt": len(newMembers),
		}).Error; err != nil {
			return errors.New("退出群聊失败")
		}
		return tx.Where("user_id = ? AND contact_id = ? AND contact_type = 1", userId, groupUuid).
			Delete(&model.UserContact{}).Error
	})
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
	newMembersJSON, _ := json.Marshal(newMembers)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&group).Updates(map[string]interface{}{
			"members":    newMembersJSON,
			"member_cnt": len(newMembers),
		}).Error; err != nil {
			return errors.New("更新群聊失败")
		}
		return tx.Where("user_id = ? AND contact_id = ? AND contact_type = 1", targetUserId, groupUuid).
			Delete(&model.UserContact{}).Error
	})
}

// 文件：back/internal/service/group.go（或你的 service 包里）
// 替换：

func DismissGroup(ownerId, groupUuid string) ([]string, error) {
	db := config.GetDB()

	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return nil, errors.New("群聊不存在")
	}
	if group.OwnerId != ownerId {
		return nil, errors.New("只有群主才能解散群聊")
	}

	// ✅ 解出成员列表，供 WS 广播使用
	var members []string
	_ = json.Unmarshal(group.Members, &members)

	err := db.Transaction(func(tx *gorm.DB) error {
		// 标记解散 & 清空成员
		group.Status = 2 // 2 = 解散
		group.Members = []byte("[]")
		group.MemberCnt = 0
		if err := tx.Save(&group).Error; err != nil {
			return err
		}
		// 删除所有成员的 user_contact 记录
		return tx.Where("contact_id = ? AND contact_type = 1", groupUuid).
			Delete(&model.UserContact{}).Error
	})
	if err != nil {
		return nil, errors.New("解散失败")
	}

	return members, nil
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

// UpdateGroupAvatar 更新群头像（群主权限）
func UpdateGroupAvatar(userId, groupUuid, avatar string) error {
	db := config.GetDB()
	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupUuid).First(&group).Error; err != nil {
		return errors.New("群聊不存在")
	}
	if group.OwnerId != userId {
		return errors.New("没有权限修改群头像")
	}
	return db.Model(&group).Update("avatar", avatar).Error
}
