package req

type RegisterRequest struct {
	//Telephone string `json:"telephone" binding:"required,len=11"`
	Nickname string `json:"nickname" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}
