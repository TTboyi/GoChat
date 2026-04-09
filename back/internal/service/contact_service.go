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
	Type     int    `json:"type"`
}

// ApplyContactByTarget 通过邮箱或ID添加联系人，返回目标用户UUID
func ApplyContactByTarget(userId, target, message string) (string, error) {
	db := config.GetDB()

	var targetUser model.UserInfo

	if strings.Contains(target, "@") {
		if err := db.Where("email = ?", target).First(&targetUser).Error; err != nil {
			return "", errors.New("该邮箱未注册用户")
		}
	} else {
		if err := db.Where("uuid = ?", target).First(&targetUser).Error; err != nil {
			return "", errors.New("该用户不存在")
		}
	}

	if targetUser.Uuid == userId {
		return "", errors.New("不能添加自己为好友")
	}

	var existing model.UserContact
	if err := db.Where("user_id = ? AND contact_id = ?", userId, targetUser.Uuid).First(&existing).Error; err == nil {
		return "", errors.New("你们已经是好友")
	}

	var pending model.ContactApply
	if err := db.
		Where("user_id = ? AND contact_id = ? AND status = 0", userId, targetUser.Uuid).
		First(&pending).Error; err == nil {
		return "", errors.New("你已发送过好友申请，请等待对方处理")
	}

	var reverse model.ContactApply
	if err := db.
		Where("user_id = ? AND contact_id = ? AND status = 0", targetUser.Uuid, userId).
		First(&reverse).Error; err == nil {
		return "", errors.New("对方已向你发送好友申请，请前往新朋友处理")
	}

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
		return "", errors.New("申请失败，请稍后再试")
	}

	return targetUser.Uuid, nil
}

// GetNewContactApplyList 获取我收到的待处理好友申请
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

// HandleContactApply 审核联系人申请，返回申请人userId（用于WS通知）
func HandleContactApply(userId string, form *req.HandleContactApplyRequest) (string, error) {
	db := config.GetDB()

	var apply model.ContactApply
	if err := db.Where("uuid = ?", form.ApplyUuid).First(&apply).Error; err != nil {
		return "", errors.New("申请不存在")
	}

	if apply.ContactId != userId {
		return "", errors.New("无权处理此申请")
	}

	if form.Approve {
		apply.Status = 1

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
			return "", err
		}
		if err := db.Create(&contact2).Error; err != nil {
			return "", err
		}
	} else {
		apply.Status = 2
	}

	return apply.UserId, db.Save(&apply).Error
}

func GetContactList(userId string) ([]ContactDetail, error) {
	db := config.GetDB()
	var contacts []model.UserContact
	if err := db.Where("user_id = ? AND status = 0", userId).Find(&contacts).Error; err != nil {
		return nil, err
	}

	var list []ContactDetail
	for _, c := range contacts {
		if c.ContactType == 0 {
			var user model.UserInfo
			if err := db.Where("uuid = ?", c.ContactId).First(&user).Error; err == nil {
				list = append(list, ContactDetail{
					Uuid:     user.Uuid,
					Nickname: user.Nickname,
					Avatar:   user.Avatar,
					Type:     0,
				})
			}
		}
	}
	return list, nil
}

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

func BlackContact(userId, targetUserId string) error {
	db := config.GetDB()
	return db.Model(&model.UserContact{}).
		Where("user_id = ? AND contact_id = ?", userId, targetUserId).
		Update("status", 1).Error
}

func UnBlackContact(userId, targetUserId string) error {
	db := config.GetDB()
	return db.Model(&model.UserContact{}).
		Where("user_id = ? AND contact_id = ?", userId, targetUserId).
		Update("status", 0).Error
}

func RefuseContactApply(userId, applyUuid string) error {
	db := config.GetDB()
	var apply model.ContactApply
	if err := db.Where("uuid = ?", applyUuid).First(&apply).Error; err != nil {
		return errors.New("申请不存在")
	}
	if apply.ContactId != userId {
		return errors.New("无权拒绝此申请")
	}
	apply.Status = 2
	return db.Save(&apply).Error
}

func BlackApply(userId, applyUuid string) error {
	db := config.GetDB()
	var apply model.ContactApply
	if err := db.Where("uuid = ?", applyUuid).First(&apply).Error; err != nil {
		return errors.New("申请不存在")
	}
	if apply.ContactId != userId {
		return errors.New("无权操作此申请")
	}
	apply.Status = 3
	return db.Save(&apply).Error
}

func GetMyJoinedGroups(userId string) ([]model.GroupInfo, error) {
	db := config.GetDB()
	var groups []model.GroupInfo

	var allGroups []model.GroupInfo
	if err := db.Find(&allGroups).Error; err != nil {
		return nil, err
	}

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

func GetContactInfo(targetId string) (interface{}, error) {
	db := config.GetDB()

	var user model.UserInfo
	if err := db.Where("uuid = ?", targetId).First(&user).Error; err == nil {
		return user, nil
	}

	var group model.GroupInfo
	if err := db.Where("uuid = ?", targetId).First(&group).Error; err == nil {
		return group, nil
	}

	return nil, errors.New("未找到该联系人")
}
