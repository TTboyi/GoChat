// ============================================================
// 文件：back/internal/controller/v1/upload.go
// 作用：处理文件上传（头像上传、图片上传、聊天文件上传）。
//
// 上传流程：
//   1. 前端通过 multipart/form-data 格式上传文件
//   2. 后端接收文件，保存到 static/files/（或 static/avatars/）目录
//   3. 返回文件的 URL 路径（相对路径，前端拼上 API_BASE 就能访问）
//   4. 前端拿到 URL 后，在发消息时把 URL 附在消息里，或更新头像字段
//
// 为什么文件上传和消息发送是两个分开的步骤（而不是一个请求）？
//   - 文件可能很大，上传是异步的，可以显示上传进度
//   - 上传成功后用户还可以取消发送（上传的文件由清理任务定期回收）
//   - 前端可以在文件上传期间让用户继续输入消息内容
// ============================================================
package v1

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"chatapp/back/internal/config"
	"chatapp/back/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// allowedImageMIME 是图片类上传允许的 MIME 类型白名单。
var allowedImageMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// allowedAvatarExt 头像允许的扩展名（与 MIME 校验双重保障）。
var allowedAvatarExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true}

// allowedImageExt 图片允许的扩展名。
var allowedImageExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}

// sanitizeFilename 去掉文件名中的路径分隔符和其他危险字符，只保留安全字符。
func sanitizeFilename(name string) string {
	name = filepath.Base(name) // 防止路径穿越
	// 只允许字母、数字、下划线、连字符、点
	re := regexp.MustCompile(`[^\w.\-]`)
	name = re.ReplaceAllString(name, "_")
	// 防止以点开头（隐藏文件）
	if strings.HasPrefix(name, ".") {
		name = "_" + name
	}
	return name
}

// sniffMIME 从文件头部字节嗅探真实 MIME 类型，不依赖客户端声明。
func sniffMIME(f interface{ Read([]byte) (int, error) }) string {
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	return http.DetectContentType(buf[:n])
}

// 上传头像
func UploadAvatar(c *gin.Context) {
	userId := c.GetString("userId")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedAvatarExt[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "头像格式只支持 jpg/jpeg/png"})
		return
	}

	// MIME 嗅探
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取文件"})
		return
	}
	defer src.Close()
	mimeType := sniffMIME(src)
	if !allowedImageMIME[strings.Split(mimeType, ";")[0]] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件内容不是有效图片"})
		return
	}

	newFileName := fmt.Sprintf("avatar_%s%s", userId, ext)
	savePath := filepath.Join(config.GetConfig().StaticAvatarPath, newFileName)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	avatarURL := fmt.Sprintf("/static/avatars/%s", newFileName)

	db := config.GetDB()
	if err := db.Model(&model.UserInfo{}).
		Where("uuid = ?", userId).
		Update("avatar", avatarURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "上传成功", "url": avatarURL})
}

// 上传图片（用于群头像等，不更新用户信息）
func UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExt[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片格式只支持 jpg/jpeg/png/gif/webp"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取文件"})
		return
	}
	defer src.Close()
	mimeType := sniffMIME(src)
	if !allowedImageMIME[strings.Split(mimeType, ";")[0]] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件内容不是有效图片"})
		return
	}

	id := strings.ReplaceAll(uuid.New().String(), "-", "")
	newFileName := fmt.Sprintf("img_%s%s", id[:12], ext)
	savePath := filepath.Join(config.GetConfig().StaticAvatarPath, newFileName)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功",
		"url":     fmt.Sprintf("/static/avatars/%s", newFileName),
	})
}

const maxFileSize = 30 * 1024 * 1024 // 30MB

// 上传文件
func UploadFile(c *gin.Context) {
	userId := c.GetString("userId")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	if file.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件大小不能超过 30MB"})
		return
	}

	// 清洗原始文件名，防止路径穿越和特殊字符注入
	safeOrigName := sanitizeFilename(file.Filename)
	ext := strings.ToLower(filepath.Ext(safeOrigName))

	// 拒绝可执行文件类型
	blockedExt := map[string]bool{
		".exe": true, ".sh": true, ".bat": true, ".cmd": true,
		".php": true, ".py": true, ".js": true, ".html": true,
		".htm": true, ".jsp": true, ".asp": true, ".aspx": true,
	}
	if blockedExt[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不允许上传该类型文件"})
		return
	}

	// MIME 嗅探：若文件头声明为文本/脚本类型则拒绝
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取文件"})
		return
	}
	buf := make([]byte, 512)
	n, _ := io.ReadFull(src, buf)
	src.Close()
	detectedMIME := http.DetectContentType(buf[:n])
	if strings.Contains(detectedMIME, "text/html") || strings.Contains(detectedMIME, "text/xml") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件内容类型不被允许"})
		return
	}

	newFileName := fmt.Sprintf("file_%s_%d%s", userId, time.Now().Unix(), ext)
	savePath := filepath.Join(config.GetConfig().StaticFilePath, newFileName)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	fileURL := fmt.Sprintf("/static/files/%s", newFileName)
	c.JSON(http.StatusOK, gin.H{"message": "上传成功", "url": fileURL})
}
