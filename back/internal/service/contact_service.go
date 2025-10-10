package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/model"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContactDetail struct {
	Uuid     string `json:"uuid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// 申请添加联系人
// ApplyContactByTarget 通过邮箱或ID添加联系人
func ApplyContactByTarget(userId, target, message string) error {
	db := config.GetDB()

	var targetUser model.UserInfo

	// 判断是邮箱还是UUID
	if strings.Contains(target, "@") {
		if err := db.Where("email = ?", target).First(&targetUser).Error; err != nil {
			return errors.New("该邮箱未注册用户")
		}
	} else {
		if err := db.Where("uuid = ?", target).First(&targetUser).Error; err != nil {
			return errors.New("该用户不存在")
		}
	}

	if targetUser.Uuid == userId {
		return errors.New("不能添加自己为好友")
	}

	// ✅ 检查是否已经是好友
	var existing model.UserContact
	if err := db.Where("user_id = ? AND contact_id = ?", userId, targetUser.Uuid).First(&existing).Error; err == nil {
		return errors.New("你们已经是好友")
	}

	// ✅ 检查是否有未处理的申请
	var pending model.ContactApply
	if err := db.
		Where("user_id = ? AND contact_id = ? AND status = 0", userId, targetUser.Uuid).
		First(&pending).Error; err == nil {
		return errors.New("你已发送过好友申请，请等待对方处理")
	}

	// ✅ 检查对方是否已向你申请
	var reverse model.ContactApply
	if err := db.
		Where("user_id = ? AND contact_id = ? AND status = 0", targetUser.Uuid, userId).
		First(&reverse).Error; err == nil {
		return errors.New("对方已向你发送好友申请，请前往“新的朋友”处理")
	}

	// ✅ 创建新的申请
	apply := model.ContactApply{
		Uuid:        "A" + uuid.NewString()[:19],
		UserId:      userId,
		ContactId:   targetUser.Uuid,
		ContactType: 0,
		Status:      0,
		Message:     message,
		LastApplyAt: time.Now(),
	}

	if err := db.Create(&apply).Error; err != nil {
		return errors.New("申请失败，请稍后再试")
	}

	return nil
}

// 获取新的联系人申请（我收到的）
type ContactApplyDetail struct {
	Uuid        string    `json:"uuid"`
	UserId      string    `json:"userId"`
	Nickname    string    `json:"nickname"`
	Avatar      string    `json:"avatar"`
	Message     string    `json:"message"`
	LastApplyAt time.Time `json:"lastApplyAt"`
}

func GetNewContactApplyList(userId string) ([]ContactApplyDetail, error) {
	db := config.GetDB()
	var list []ContactApplyDetail

	err := db.Table("contact_apply AS a").
		Select("a.uuid, a.user_id, u.nickname, u.avatar, a.message, a.last_apply_at").
		Joins("JOIN user_info AS u ON a.user_id = u.uuid").
		Where("a.contact_id = ? AND a.contact_type = 0 AND a.status = 0", userId).
		Order("a.last_apply_at DESC").
		Scan(&list).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if list == nil {
		list = []ContactApplyDetail{}
	}
	return list, nil
}

// 审核联系人申请
func HandleContactApply(userId string, form *req.HandleContactApplyRequest) error {
	db := config.GetDB()

	var apply model.ContactApply
	if err := db.Where("uuid = ?", form.ApplyUuid).First(&apply).Error; err != nil {
		return errors.New("申请不存在")
	}

	if apply.ContactId != userId {
		return errors.New("无权处理此申请")
	}

	if form.Approve {
		apply.Status = 1 // 通过

		// 在 user_contact 中互相加好友
		contact1 := model.UserContact{
			UserId:      userId,
			ContactId:   apply.UserId,
			ContactType: 0,
			Status:      0,
			CreatedAt:   time.Now(),
		}
		contact2 := model.UserContact{
			UserId:      apply.UserId,
			ContactId:   userId,
			ContactType: 0,
			Status:      0,
			CreatedAt:   time.Now(),
		}
		if err := db.Create(&contact1).Error; err != nil {
			return err
		}
		if err := db.Create(&contact2).Error; err != nil {
			return err
		}

	} else {
		apply.Status = 2 // 拒绝
	}

	return db.Save(&apply).Error
}

func GetContactList(userId string) ([]ContactDetail, error) {
	db := config.GetDB()
	var contacts []model.UserContact
	if err := db.Where("user_id = ? AND status = 0", userId).Find(&contacts).Error; err != nil {
		return nil, err
	}

	var list []ContactDetail
	for _, c := range contacts {
		var user model.UserInfo
		if err := db.Where("uuid = ?", c.ContactId).First(&user).Error; err == nil {
			list = append(list, ContactDetail{
				Uuid:     user.Uuid,
				Nickname: user.Nickname,
				Avatar:   user.Avatar,
			})
		}
	}
	return list, nil
}

// 删除联系人（双向删除）
func DeleteContact(userId, targetUserId string) error {
	db := config.GetDB()
	if err := db.Where("user_id = ? AND contact_id = ?", userId, targetUserId).Delete(&model.UserContact{}).Error; err != nil {
		return err
	}
	if err := db.Where("user_id = ? AND contact_id = ?", targetUserId, userId).Delete(&model.UserContact{}).Error; err != nil {
		return err
	}
	return nil
}

// 拉黑联系人
func BlackContact(userId, targetUserId string) error {
	db := config.GetDB()
	return db.Model(&model.UserContact{}).
		Where("user_id = ? AND contact_id = ?", userId, targetUserId).
		Update("status", 1).Error // 1 = 黑名单
}

// 解除拉黑联系人
func UnBlackContact(userId, targetUserId string) error {
	db := config.GetDB()
	return db.Model(&model.UserContact{}).
		Where("user_id = ? AND contact_id = ?", userId, targetUserId).
		Update("status", 0).Error // 0 = 正常
}

// 拒绝联系人申请
func RefuseContactApply(userId, applyUuid string) error {
	db := config.GetDB()

	var apply model.ContactApply
	if err := db.Where("uuid = ?", applyUuid).First(&apply).Error; err != nil {
		return errors.New("申请不存在")
	}
	if apply.ContactId != userId {
		return errors.New("无权拒绝此申请")
	}
	apply.Status = 2 // 拒绝
	return db.Save(&apply).Error
}

// 拉黑联系人申请
func BlackApply(userId, applyUuid string) error {
	db := config.GetDB()

	var apply model.ContactApply
	if err := db.Where("uuid = ?", applyUuid).First(&apply).Error; err != nil {
		return errors.New("申请不存在")
	}
	if apply.ContactId != userId {
		return errors.New("无权操作此申请")
	}
	apply.Status = 3 // 拉黑
	return db.Save(&apply).Error
}

// 获取我加入的群聊
func GetMyJoinedGroups(userId string) ([]model.GroupInfo, error) {
	db := config.GetDB()
	var groups []model.GroupInfo

	// 查询所有群聊
	var allGroups []model.GroupInfo
	if err := db.Find(&allGroups).Error; err != nil {
		return nil, err
	}

	// 过滤出包含 userId 的群
	for _, g := range allGroups {
		var members []string
		if len(g.Members) > 0 {
			_ = json.Unmarshal(g.Members, &members)
		}
		for _, m := range members {
			if m == userId {
				groups = append(groups, g)
				break
			}
		}
	}

	return groups, nil
}

// 获取联系人信息（可以是用户或群）
func GetContactInfo(targetId string) (interface{}, error) {
	db := config.GetDB()

	// 先查用户
	var user model.UserInfo
	if err := db.Where("uuid = ?", targetId).First(&user).Error; err == nil {
		return user, nil
	}

	// 再查群聊
	var group model.GroupInfo
	if err := db.Where("uuid = ?", targetId).First(&group).Error; err == nil {
		return group, nil
	}

	return nil, errors.New("未找到该联系人")
}
