package resp

// UserInfoResponse 前端展示的用户信息
type UserInfoResponse struct {
	Uuid      string `json:"uuid"`
	Nickname  string `json:"nickname"`
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Signature string `json:"signature"`
	IsAdmin   int8   `json:"isAdmin"`
}
