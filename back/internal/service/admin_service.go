package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/model"
	"errors"
	"time"
)

func GetAllUsers() ([]model.UserInfo, error) {
	db := config.GetDB()
	var users []model.UserInfo
	if err := db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func BanUser(userId string, status bool) error {
	db := config.GetDB()
	var user model.UserInfo
	if err := db.Where("uuid = ?", userId).First(&user).Error; err != nil {
		return errors.New("用户不存在")
	}
	if status {
		user.Status = 1 // 1 = 封禁
	} else {
		user.Status = 0 // 0 = 正常
	}
	return db.Save(&user).Error
}

func GetAllGroups() ([]model.GroupInfo, error) {
	db := config.GetDB()
	var groups []model.GroupInfo
	if err := db.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func AdminDismissGroup(groupId string) error {
	db := config.GetDB()
	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupId).First(&group).Error; err != nil {
		return errors.New("群聊不存在")
	}
	group.Status = 2 // 解散
	group.Members = []byte("[]")
	group.MemberCnt = 0
	return db.Save(&group).Error
}

// SystemStats 是 /admin/stats 的返回结构，字段含义见各注释。
type SystemStats struct {
	// 累计总量
	TotalUsers    int64 `json:"total_users"`
	TotalGroups   int64 `json:"total_groups"`
	TotalMessages int64 `json:"total_messages"`
	// 今日增量
	TodayMessages int64 `json:"today_messages"`
	TodayNewUsers int64 `json:"today_new_users"`
	// 消息类型分布
	TextMessages int64 `json:"text_messages"`
	FileMessages int64 `json:"file_messages"`
	// 实时在线（由调用方注入，service 层不直接引用 chat 包避免循环依赖）
	OnlineUsers int64 `json:"online_users"`
}

// DailyStat 是 /admin/stats/daily 单天数据点。
type DailyStat struct {
	Date     string `json:"date"`
	Messages int64  `json:"messages"`
	NewUsers int64  `json:"new_users"`
}

func GetSystemStats(onlineUsers int64) (SystemStats, error) {
	db := config.GetDB()
	today := time.Now().Format("2006-01-02")

	var s SystemStats
	s.OnlineUsers = onlineUsers

	db.Model(&model.UserInfo{}).Count(&s.TotalUsers)
	db.Model(&model.GroupInfo{}).Count(&s.TotalGroups)
	db.Model(&model.Message{}).Count(&s.TotalMessages)

	db.Model(&model.Message{}).Where("DATE(created_at) = ?", today).Count(&s.TodayMessages)
	db.Model(&model.UserInfo{}).Where("DATE(created_at) = ?", today).Count(&s.TodayNewUsers)

	db.Model(&model.Message{}).Where("type = 0").Count(&s.TextMessages)
	db.Model(&model.Message{}).Where("type = 1").Count(&s.FileMessages)

	return s, nil
}

// GetDailyStats 查询最近 days 天每日消息数和新增用户数（供折线图使用）。
func GetDailyStats(days int) ([]DailyStat, error) {
	db := config.GetDB()
	if days <= 0 || days > 90 {
		days = 7
	}

	result := make([]DailyStat, days)
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dateStr := day.Format("2006-01-02")
		idx := days - 1 - i

		result[idx].Date = dateStr

		db.Model(&model.Message{}).
			Where("DATE(created_at) = ?", dateStr).
			Count(&result[idx].Messages)

		db.Model(&model.UserInfo{}).
			Where("DATE(created_at) = ?", dateStr).
			Count(&result[idx].NewUsers)
	}

	return result, nil
}

