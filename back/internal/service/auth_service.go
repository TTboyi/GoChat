package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/utils"
	"context"
	"strings"
	"time"
)

// 刷新 token
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

// 退出登录：将 access token 放入黑名单直到它自然过期
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
