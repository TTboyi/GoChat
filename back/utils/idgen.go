package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mrand "math/rand"
	"time"
)

// init 用于初始化 math/rand 的随机种子（仅用于生成数字型 UUID）
func init() {
	mrand.Seed(time.Now().UnixNano())
}

// GenerateUserUUID 生成 8 位纯数字用户 UUID
// 示例: 02468135
func GenerateUserUUID() string {
	return fmt.Sprintf("%08d", mrand.Intn(100000000))
}

// GenerateGroupUUID 生成 6 位纯数字群 UUID
// 示例: 528413
func GenerateGroupUUID() string {
	return fmt.Sprintf("%06d", mrand.Intn(1000000))
}

// GenerateUUID 生成指定长度的随机字符串（安全版）
// 例如: GenerateUUID(20) -> "a8f2Ck19pQx9s0Yw3hTz"
func GenerateUUID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range result {
		// 使用 crypto/rand 生成加密安全随机数
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			// 如果失败，退回 'a'
			result[i] = 'a'
			continue
		}
		result[i] = charset[n.Int64()]
	}

	return string(result)
}
