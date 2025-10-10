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
