// ============================================================
// 文件：back/internal/service/auth_service.go
// 作用：实现 token 刷新和登出的业务逻辑。
//
// RefreshToken 的安全设计：
//   不只是简单地让 JWT 库刷新 token，还要先检查 Redis 黑名单：
//   如果 access token 已经被登出放进黑名单，即使 refresh token 还有效，
//   也拒绝刷新。这防止了"用户已登出，但攻击者用盗来的 refresh token 重新获取 access"的场景。
//
// Logout 的核心思想：
//   JWT 的天然缺陷是"签发后无法撤销"。
//   解决方案：把 token 加入 Redis 黑名单，并设置与 token 剩余有效期相同的 TTL。
//   TTL 到期时，token 本身也已过期，黑名单记录自动被 Redis 删除，不占用额外空间。
//   这叫"黑名单+TTL"的无状态会话管理策略。
// ============================================================
package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/utils"
	"context"
	"strings"
	"time"
)

// RefreshToken 校验旧 token 状态后，签发一组新的 access/refresh token。
// 这里额外检查 Redis 黑名单，避免已经“逻辑登出”的 access token 还能继续刷新。
func RefreshToken(accessToken, refreshToken string) (string, string, error) {
	j := utils.GetJWT()
	// 如果 access 在黑名单里，直接不允许刷新
	rdb := config.GetRedis()
	ctx := context.Background()
	if exists, _ := rdb.Exists(ctx, "jwt:blacklist:"+accessToken).Result(); exists > 0 {
		return "", "", utils.ErrTokenInvalid
	}
	return j.RefreshToken(accessToken, refreshToken)
}

// Logout 的核心思想不是立即删除 JWT，而是把它放进 Redis 黑名单直到自然过期。
// 因为 JWT 一旦签发就无法真正“撤销”，所以很多系统都会用黑名单来补齐登出语义。
func Logout(accessToken string) error {
	j := utils.GetJWT()

	// 解析拿到过期时间
	claims, err := j.ParseAccessToken(strings.TrimPrefix(accessToken, "Bearer "))
	if err != nil {
		// 已失效就算成功：无须入黑名单
		return nil
	}
	exp := time.Until(claims.ExpiresAt.Time)
	if exp <= 0 {
		return nil
	}

	rdb := config.GetRedis()
	ctx := context.Background()
	key := "jwt:blacklist:" + strings.TrimPrefix(accessToken, "Bearer ")
	// value 无所谓，设置 TTL 即可
	return rdb.Set(ctx, key, "1", exp).Err()
}
