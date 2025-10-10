package v1

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"chatapp/back/internal/config"
	"chatapp/back/internal/model"

	"github.com/gin-gonic/gin"
)

// 上传头像
func UploadAvatar(c *gin.Context) {
	userId := c.GetString("userId") // JWT 注入的用户ID
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "头像格式只支持 jpg/jpeg/png"})
		return
	}

	// 新文件名: avatar_<userId>.ext
	newFileName := fmt.Sprintf("avatar_%s%s", userId, ext)
	savePath := filepath.Join(config.GetConfig().StaticAvatarPath, newFileName)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	// 生成可访问 URL
	avatarURL := fmt.Sprintf("/static/avatars/%s", newFileName)

	// ✅ 更新数据库 user_info.avatar 字段
	db := config.GetDB()
	if err := db.Model(&model.UserInfo{}).
		Where("uuid = ?", userId).
		Update("avatar", avatarURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功",
		"url":     avatarURL,
	})
}

// 上传文件
func UploadFile(c *gin.Context) {
	userId := c.GetString("userId")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	// 保留原始扩展名
	ext := filepath.Ext(file.Filename)
	newFileName := fmt.Sprintf("file_%s_%d%s", userId, time.Now().Unix(), ext)
	savePath := filepath.Join(config.GetConfig().StaticFilePath, newFileName)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	fileURL := fmt.Sprintf("/static/files/%s", newFileName)
	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功",
		"url":     fileURL,
	})
}
