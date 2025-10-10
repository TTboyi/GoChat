package v1

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/model"
	"chatapp/back/internal/service"
	"chatapp/back/utils"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Register 处理用户注册
func Register(c *gin.Context) {
	var form req.RegisterRequest

	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误"})
		return
	}
	uuid8 := generateUserID()
	// 构造 model.UserInfo（把 req 映射进去）
	user := &model.UserInfo{
		Uuid:      uuid8,
		Telephone: "",
		Password:  form.Password, // 可加密：utils.Encrypt(form.Password)
		Nickname:  form.Nickname,
		CreatedAt: time.Now(),
		IsAdmin:   0,
		Status:    0,
	}

	db := config.GetDB()

	if err := service.RegisterUser(db, user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
	})
}

// generateShortID 生成 8 位随机数字字符串
func generateUserID() string {
	for {
		id := fmt.Sprintf("%08d", rand.Intn(100000000))
		var count int64
		db := config.GetDB()
		db.Model(&model.UserInfo{}).Where("uuid = ?", id).Count(&count)
		if count == 0 {
			return id
		}
	} // 0~99999999
}

func Login(c *gin.Context) {
	var form req.LoginRequest
	fmt.Println("JWT密钥:", string(utils.GetJWT().Key))

	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误"})
		return
	}

	token, refresh, err := service.LoginUser(config.GetDB(), form.Nickname, form.Password, utils.GetJWT())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "登录成功",
		"token":   token,
		"refresh": refresh,
	})
}

func GetUserInfo(c *gin.Context) {
	// JWT 中间件已经注入了 userId
	userId := c.GetString("userId")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	user, err := service.GetUserInfo(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

// UpdateUserInfo 修改用户资料
func UpdateUserInfo(c *gin.Context) {
	userId := c.GetString("userId")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var form req.UpdateUserRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	user, err := service.UpdateUserInfo(userId, &form)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": user})
}
