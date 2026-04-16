// ============================================================
// 文件：back/utils/jwt.go
// 作用：JWT（JSON Web Token）工具封装，提供 token 生成、解析、刷新能力。
//
// 什么是 JWT？
//   JWT 是一种"无状态身份令牌"。登录成功后，服务器给用户颁发一个 JWT，
//   此后用户每次请求都携带这个 token，服务器通过验证签名来确认身份，
//   不需要在服务器端保存任何会话状态（对比传统 Session：服务器要存每个用户的登录状态）。
//
// JWT 的结构（三段 Base64 用"."拼接）：
//   header.payload.signature
//   - header: 算法类型（这里用 HS256 = HMAC-SHA256）
//   - payload: 携带的数据（userID、过期时间、签发者等）
//   - signature: 用 secret key 对前两段的签名，防篡改
//
// 双 token 设计（access + refresh）：
//   access token：有效期短（60分钟），每次 API 请求携带，过期需要刷新
//   refresh token：有效期长（1440分钟=1天），专门用于换取新的 access token
//   这样设计的好处：
//   - access token 短有效期，即使泄露，攻击者能用的时间窗口很短
//   - refresh token 长有效期，用户不需要频繁重新登录
//
// 黑名单机制（在 service/auth_service.go 里配合使用）：
//   JWT 本身无法"撤销"：一旦签发，在过期前都是有效的。
//   解决方案：登出时把这个 token 放进 Redis 黑名单（设置和 token 相同的过期时间），
//   中间件在验证 token 时额外检查黑名单，在黑名单里的 token 拒绝访问。
// ============================================================
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
