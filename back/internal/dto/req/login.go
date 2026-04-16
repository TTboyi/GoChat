// ============================================================
// 文件：back/internal/dto/req/login.go
// 作用：登录请求的参数结构体。binding:"required" 标签让 Gin 自动校验字段是否为空，
//       缺失时直接返回 400 Bad Request。
// ============================================================
package req

type LoginRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}
