// utils.jwt.go
package utils

import (
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// =================== //
// 🔐 自定义声明结构体
// =================== //
type JWTCustomClaims struct {
	UserID  string `json:"userId"`
	IsAdmin int8   `json:"role"` // "user" 普通用户，"admin" 管理员
	jwt.RegisteredClaims
}

// =================== //
// 🔐 错误定义
// =================== //
var (
	ErrTokenGenFailed         = errors.New("令牌生成失败")
	ErrTokenExpired           = errors.New("令牌已过期")
	ErrTokenExpiredMaxRefresh = errors.New("令牌已过最大刷新时间")
	ErrTokenMalformed         = errors.New("令牌格式错误")
	ErrTokenInvalid           = errors.New("令牌无效")
	ErrTokenNotFound          = errors.New("未提供令牌")
)

// =================== //
// 🔐 核心结构体
// =================== //
type ARJWT struct {
	Key               []byte
	AccessExpireTime  int64 // access token 过期时间（分钟）
	RefreshExpireTime int64 // refresh token 过期时间（分钟）
	Issuer            string
}

// =================== //
// 🔐 构造函数
// =================== //
func NewARJWT(secret, issuer string, accessExpireTime, refreshExpireTime int64) *ARJWT {
	if refreshExpireTime <= accessExpireTime {
		log.Fatal("refresh token 过期时间必须大于 access token")
	}
	return &ARJWT{
		Key:               []byte(secret),
		AccessExpireTime:  accessExpireTime,
		RefreshExpireTime: refreshExpireTime,
		Issuer:            issuer,
	}
}

// =================== //
// 🔐 生成 token
// =================== //
func (j *ARJWT) GenerateToken(userID string, isAdmin int8) (accessToken, refreshToken string, err error) {
	now := time.Now()

	// access token 带 userId 和 role
	accessClaims := JWTCustomClaims{
		UserID:  userID,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(j.AccessExpireTime) * time.Minute)),
			Issuer:    j.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(j.Key)
	if err != nil {
		return "", "", ErrTokenGenFailed
	}

	// refresh token 不需要带 userId 和 role
	refreshClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(j.RefreshExpireTime) * time.Minute)),
		Issuer:    j.Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
	}

	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(j.Key)
	if err != nil {
		return "", "", ErrTokenGenFailed
	}

	return accessToken, refreshToken, nil
}

// =================== //
// 🔐 校验 access token
// =================== //
func (j *ARJWT) ParseAccessToken(tokenString string) (*JWTCustomClaims, error) {
	claims := new(JWTCustomClaims)

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return j.Key, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrTokenMalformed
		}
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// =================== //
// 🔁 刷新 token
// =================== //
func (j *ARJWT) RefreshToken(accessToken, refreshToken string) (newAccessToken, newRefreshToken string, err error) {
	// 检查 refresh token 是否还有效
	if _, err = jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return j.Key, nil
	}); err != nil {
		return "", "", ErrTokenExpiredMaxRefresh
	}

	// 尝试解析 access token（即使它过期）
	var claims JWTCustomClaims
	_, err = jwt.ParseWithClaims(accessToken, &claims, func(token *jwt.Token) (interface{}, error) {
		return j.Key, nil
	})

	if err != nil {

		if errors.Is(err, jwt.ErrTokenExpired) {
			// access token 确实过期，但 refresh 还没 → 重新签发
			return j.GenerateToken(claims.UserID, claims.IsAdmin)
		}
	}

	// access token 没有过期，不允许刷新
	return "", "", ErrTokenInvalid
}

var jwtInstance *ARJWT

// InitJWT 初始化全局 JWT 实例
func InitJWT(secret, issuer string, accessExpireTime, refreshExpireTime int64) {
	if jwtInstance != nil {
		log.Println("⚠️ JWT 已初始化，重复调用将被忽略")
		return
	}
	jwtInstance = NewARJWT(secret, issuer, accessExpireTime, refreshExpireTime)
}

// GetJWT 返回全局 JWT 实例
func GetJWT() *ARJWT {
	if jwtInstance == nil {
		log.Fatal("JWT 未初始化，请先调用 InitJWT")
	}
	return jwtInstance
}
