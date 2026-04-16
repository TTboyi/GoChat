// ============================================================
// 文件：back/internal/dto/resp/user_resp.go
// 作用：用户信息查询的响应数据结构体（UserInfoResponse）。
//       故意省略了 Password 字段，确保接口永远不会把密码返回给前端。
//       这是"最小权限"原则在接口设计上的体现。
// ============================================================
package resp

// UserInfoResponse 前端展示的用户信息
type UserInfoResponse struct {
	Uuid      string `json:"uuid"`
	Nickname  string `json:"nickname"`
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Signature string `json:"signature"`
	IsAdmin   int8   `json:"is_admin"`
}
