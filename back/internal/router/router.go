package router

import (
	v1 "chatapp/back/internal/controller/v1"
	"chatapp/back/internal/middleware"
	"chatapp/back/utils"

	//"time"

	//"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
)

func initMessageRoutes(r *gin.RouterGroup) {
	message := r.Group("/message")
	{
		message.POST("/list", v1.GetMessageList)           // 获取私聊消息
		message.POST("/groupList", v1.GetGroupMessageList) // 获取群聊消息
		message.POST("/uploadAvatar", v1.UploadAvatar)     // 上传头像
		message.POST("/uploadFile", v1.UploadFile)         // 上传文件
	}

}

func initUserRoutes(r *gin.RouterGroup) {
	user := r.Group("/user")
	{
		user.POST("/update", v1.UpdateUserInfo)

	}
}

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
		group.GET("/info", v1.GetGroupInfo)

	}
}

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

// ================================ //
// 🌐 WebSocket 路由
func initWsRoutes(r *gin.RouterGroup) {
	//r.GET("/wss", v1.WsLogin)
	//r.POST("/wsLogout", v1.WsLogout) // 以后我们实现登出
}

// InitRouter 初始化路由，仅包含注册接口和 WebSocket 登录
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
	r.Static("/static/avatars", "./static/avatars")
	r.Static("/static/files", "./static/files")

	// 注册接口
	r.POST("/register", v1.Register)
	r.POST("/auth/refresh", v1.RefreshToken)
	r.POST("/auth/logout", v1.Logout)
	r.POST("/login", v1.Login)
	r.GET("/wss", v1.WsLogin)
	r.POST("/wss", v1.WsLogout)
	r.POST("/captcha/send_email", v1.SendEmailCaptcha)
	r.POST("/captcha/login_email", v1.EmailCaptchaLogin)

	// WebSocket 登录（如暂未实现，可先注释）
	//r.GET("/wss", v1.WsLogin)

	// 鉴权保护接口（当前无内容，可以删除或保留空组）
	authGroup := r.Group("/", middleware.AuthMiddleware(jwt)) // 占位
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
