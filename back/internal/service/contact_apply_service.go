// ============================================================
// 文件：back/internal/service/contact_apply_service.go
// 作用：群聊申请（进群申请）的处理逻辑。
//
// 与 contact_service.go 的区别：
//   contact_service.go 处理"好友申请"（用户间的一对一关系）
//   contact_apply_service.go 处理"入群申请"（用户申请加入某个群）
//
// 群聊审核模式（AddMode = 1）：
//   群主或管理员在这里审核待加入的申请。
//   通过后，被申请者的 UUID 会被 append 进群的 members JSON 数组，
//   同时在 user_contact 表创建记录。
// ============================================================
package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/model"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// 新建一条加群申请
func CreateGroupApply(userId, groupId, message string) error {
	db := config.GetDB()

	apply := model.ContactApply{
		Uuid:        "A" + uuid.NewString()[:19], // 20字符以内
		UserId:      userId,
		ContactId:   groupId,
		ContactType: 1, // 1 表示群聊
		Status:      0, // 申请中
		Message:     message,
		LastApplyAt: time.Now(),
	}

	return db.Create(&apply).Error
}

// 查询某个群聊的待审核申请
func GetGroupApplyList(groupId string) ([]model.ContactApply, error) {
	db := config.GetDB()
	var list []model.ContactApply
	if err := db.Where("contact_id = ? AND status = 0", groupId).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// 审核加群申请（通过/拒绝）
func HandleGroupApply(applyUuid string, approve bool) error {
	db := config.GetDB()

	var apply model.ContactApply
	if err := db.Where("uuid = ?", applyUuid).First(&apply).Error; err != nil {
		return errors.New("申请不存在")
	}

	// 如果是通过，更新群聊成员
	if approve {
		var group model.GroupInfo
		if err := db.Where("uuid = ?", apply.ContactId).First(&group).Error; err != nil {
			return errors.New("群聊不存在")
		}

		// 解析已有成员
		var members []string
		if len(group.Members) > 0 {
			_ = json.Unmarshal(group.Members, &members)
		}

		// 检查是否已经在群里
		for _, m := range members {
			if m == apply.UserId {
				return errors.New("用户已在群聊中")
			}
		}

		// 添加新成员
		members = append(members, apply.UserId)
		newMembers, _ := json.Marshal(members)

		group.Members = newMembers
		group.MemberCnt = len(members)

		if err := db.Save(&group).Error; err != nil {
			return errors.New("更新群聊失败")
		}

		apply.Status = 1 // 通过
	} else {
		apply.Status = 2 // 拒绝
	}

	return db.Save(&apply).Error
}
