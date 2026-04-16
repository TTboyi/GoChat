// ============================================================
// 文件：back/utils/radis.go（注：文件名有拼写错误，应为 redis.go）
// 作用：封装 Redis 客户端的初始化和验证码相关操作。
//
// 全局变量 Rdb：
//   是 redis.Client 的全局单例，整个程序共享。
//   Redis 客户端内部维护了连接池，并发安全，可以在多个 goroutine 里直接用。
//
// 验证码相关函数：
//   SaveCaptcha(key, code)：以 "captcha:{key}" 为 Redis key，
//     存入验证码并设置 5 分钟过期（TTL = 5 * time.Minute）。
//     TTL（Time To Live，存活时间）到期后 Redis 自动删除，不需要手动清理。
//   GetCaptcha(key)：读取 "captcha:{key}"，
//     如果已过期或不存在，返回 redis.Nil 错误（前端会提示"验证码已过期"）。
//
// 为什么验证码存 Redis 而不是 MySQL？
//   - 验证码有时效性（5分钟），适合用 TTL 自动清理
//   - 验证码读写极其频繁（每次发送、每次验证都要操作）
//   - Redis 的键值操作比 MySQL 快几十到上百倍
// ============================================================
package utils

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var ctx = context.Background()

func InitRedis(addr, password string, db int) {
	Rdb = redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
}

func SaveCaptcha(key, code string) error {
	return Rdb.Set(ctx, "captcha:"+key, code, 5*time.Minute).Err()
}

func GetCaptcha(key string) (string, error) {
	return Rdb.Get(ctx, "captcha:"+key).Result()
}
