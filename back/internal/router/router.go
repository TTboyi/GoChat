package router

import (
	v1 "chatapp/back/internal/controller/v1"
	"chatapp/back/internal/middleware"
	"chatapp/back/utils"

	//"time"

	//"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
)

// initMessageRoutes 注册消息相关接口。
// 阅读时可以把它理解成“聊天内容的 HTTP 补充通道”：
// 真正的实时消息走 WebSocket，而历史记录、文件上传、撤回、已读等走 REST。
func initMessageRoutes(r *gin.RouterGroup) {
	message := r.Group("/message")
	{
		message.POST("/list", v1.GetMessageList)                   // 获取私聊消息
		message.POST("/groupList", v1.GetGroupMessageList)         // 获取群聊消息
		message.POST("/uploadAvatar", v1.UploadAvatar)             // 上传头像（更新用户profile）
		message.POST("/uploadImage", v1.UploadImage)               // 上传图片（群头像等，不绑定用户）
		message.POST("/uploadFile", v1.UploadFile)                 // 上传文件
		message.POST("/recall", v1.RecallMessageFull)              // 撤回消息
		message.POST("/markRead", v1.MarkMessagesRead)             // 标记已读
		message.POST("/clearConversation", v1.ClearConversation)   // 删除好友时清除聊天记录
	}

}

// initUserRoutes 注册用户资料维护接口。
func initUserRoutes(r *gin.RouterGroup) {
	user := r.Group("/user")
	{
		user.POST("/update", v1.UpdateUserInfo)

	}
}

// initAdminRoutes 注册管理员专用接口。
// 这里是二次保护：先通过普通登录鉴权，再通过 AdminOnly 校验管理员身份。
func initAdminRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/api", middleware.AuthMiddleware(utils.GetJWT()))
	{
		auth.GET("/user/info", v1.GetUserInfo)
		admin := rg.Group("/admin", middleware.AdminOnly())
		{
			admin.GET("/users", v1.GetAllUsers)
			admin.PUT("/users/:id/ban", v1.BanUser)
			admin.GET("/groups", v1.GetAllGroups)
			admin.DELETE("/groups/:id", v1.AdminDismissGroup)
			admin.GET("/stats", v1.GetSystemStats)
		}
	}
}

// initGroupRoutes 注册群聊生命周期相关接口。
func initGroupRoutes(r *gin.RouterGroup) {
	group := r.Group("/group")
	{
		group.POST("/create", v1.CreateGroup)             // 创建群聊
		group.GET("/loadMyGroup", v1.LoadMyGroup)         // 查询我创建的群聊
		group.GET("/checkAddMode", v1.CheckGroupAddMode)  // 检查加群方式
		group.POST("/enter", v1.EnterGroupDirectly)       // 加入群聊（直接/申请）
		group.POST("/quit", v1.QuitGroup)                 // 退出群聊
		group.GET("/members", v1.GetGroupMemberList)      // 查询群聊成员列表
		group.POST("/removeMember", v1.RemoveGroupMember) // 移除群成员
		group.POST("/dismiss", v1.DismissGroupHandler)    // 解散群聊
		group.POST("/updateName", v1.UpdateGroupName)
		group.POST("/updateNotice", v1.UpdateGroupNotice)
		group.POST("/updateAvatar", v1.UpdateGroupAvatar)
		group.GET("/info", v1.GetGroupInfo)

	}
}

// initContactRoutes 管理好友申请、联系人关系和入群申请。
func initContactRoutes(r *gin.RouterGroup) {
	apply := r.Group("/apply")
	{
		apply.POST("/createGroupApply", v1.CreateGroupApply)  // 提交入群申请
		apply.GET("/getGroupApplyList", v1.GetGroupApplyList) // 查看群聊待审核申请
		apply.POST("/handleGroupApply", v1.HandleGroupApply)  // 审核通过/拒绝
	}

	contact := r.Group("/contact")
	{
		contact.POST("/apply", v1.ApplyContact)             // 申请添加联系人
		contact.GET("/newList", v1.GetNewContactList)       // 获取新的联系人申请
		contact.POST("/handle", v1.HandleContactApply)      // 审核申请（通过/拒绝）
		contact.GET("/list", v1.GetContactList)             // 获取联系人列表
		contact.POST("/delete", v1.DeleteContact)           // 删除联系人
		contact.POST("/black", v1.BlackContact)             // 拉黑联系人
		contact.POST("/unblack", v1.UnBlackContact)         // 解除拉黑联系人
		contact.POST("/refuseApply", v1.RefuseContactApply) // 拒绝联系人申请
		contact.POST("/blackApply", v1.BlackApply)          // 拉黑联系人申请
		contact.GET("/joinedGroups", v1.LoadMyJoinedGroup)  // 获取我加入的群聊
		contact.POST("/info", v1.GetContactInfo)            // 获取联系人信息

	}

}

// initSessionRoutes 注册“会话列表”相关接口。
// 这里的 session 更接近“聊天列表项”，不是 HTTP session。
func initSessionRoutes(r *gin.RouterGroup) {
	session := r.Group("/session")
	{
		session.POST("/open", v1.OpenSession)                    // 打开会话
		session.GET("/userList", v1.GetUserSessionList)          // 获取用户会话列表
		session.GET("/groupList", v1.GetGroupSessionList)        // 获取群聊会话列表
		session.POST("/delete", v1.DeleteSession)                // 删除会话
		session.GET("/checkAllowed", v1.CheckOpenSessionAllowed) // 检查是否允许打开
	}

}

// initWsRoutes WebSocket 及 TURN 相关路由
func initWsRoutes(r *gin.RouterGroup) {
	// TURN 凭证接口虽然不是 WebSocket 本身，
	// 但它直接服务于 WebRTC 通话，因此放在同一组里方便理解。
	turn := r.Group("/turn")
	{
		turn.GET("/credentials", v1.GetTurnCredentials)
	}
}

// InitRouter 统一组装整个 Gin 路由树。
// 建议阅读顺序：
// 1. 先看公开接口（登录、注册、刷新 token、WebSocket 握手）；
// 2. 再看 authGroup 里受保护的业务接口；
// 3. 最后结合 controller/service 追具体业务实现。
func InitRouter() *gin.Engine {
	jwt := utils.GetJWT()
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// r.Use(cors.New(cors.Config{
	// 	AllowOrigins:     []string{"http://localhost:5173"}, // React前端端口
	// 	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	// 	AllowHeaders:     []string{"Content-Type", "Authorization"},
	// 	AllowCredentials: true,
	// 	MaxAge:           12 * time.Hour,
	// }))

	// 静态资源
	// 前端上传后的头像/文件最终都会经由这两个目录对外暴露。
	r.Static("/static/avatars", "./static/avatars")
	r.Static("/static/files", "./static/files")

	// 公开接口：无需先走 AuthMiddleware。
	r.POST("/register", v1.Register)
	r.POST("/auth/refresh", v1.RefreshToken)
	r.POST("/auth/logout", v1.Logout)
	r.POST("/login", v1.Login)
	r.GET("/wss", v1.WsLogin)
	r.POST("/wss", v1.WsLogout)
	r.POST("/captcha/send_email", v1.SendEmailCaptcha)
	r.POST("/captcha/login_email", v1.EmailCaptchaLogin)

	// 鉴权保护接口：下面的分组都要求先携带 access token。
	authGroup := r.Group("/", middleware.AuthMiddleware(jwt))
	{
		initGroupRoutes(authGroup)
		initContactRoutes(authGroup)
		initSessionRoutes(authGroup)
		initMessageRoutes(authGroup)
		initWsRoutes(authGroup)
		initAdminRoutes(authGroup) // 只有登录后并且管理员用户能访问
		initUserRoutes(authGroup)

	}

	return r
}
