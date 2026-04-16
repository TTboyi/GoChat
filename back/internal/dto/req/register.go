// ============================================================
// 文件：back/internal/dto/req/register.go
// 作用：用户注册请求的参数结构体。
// ============================================================
package req

type RegisterRequest struct {
	//Telephone string `json:"telephone" binding:"required,len=11"`
	Nickname string `json:"nickname" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}
