// ============================================================
// 文件：back/internal/dto/req/user_req.go
// 作用：用户信息更新请求的参数结构体。所有字段可选，只更新非空字段。
// ============================================================
package req

// UpdateUserRequest 修改用户资料
type UpdateUserRequest struct {
	Nickname  string `json:"nickname"`  // 昵称
	Email     string `json:"email"`     // 邮箱
	Avatar    string `json:"avatar"`    // 头像 URL
	Signature string `json:"signature"` // 个性签名
	Password  string `json:"password"`  // 新密码（可选）
}
