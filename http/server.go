package http

import (
	"fmt"

	"IM/http/User"

	"github.com/gin-gonic/gin"
)

// Server HTTP 服务器，持有监听地址与端口。
type Server struct {
	address string // 监听地址
	port    int    // 监听端口
}

// NewServer 创建 HTTP 服务器实例。
func NewServer(address string, port int) *Server {
	return &Server{
		address: address,
		port:    port,
	}
}

// Start 注册路由并启动 HTTP 服务，启动失败时 panic。
func (s *Server) Start() {
	router := gin.Default()

	api := router.Group("/api")
	{
		// 无需鉴权的接口：注册与登录
		user := api.Group("/user")
		{
			user.POST("/register", User.Register)
			user.POST("/login", User.Login)
		}

		// 需要鉴权的接口，统一挂载 AuthMiddleware
		auth := api.Group("", User.AuthMiddleware())
		{
			auth.GET("/user/profile", User.GetProfile)
			auth.PUT("/user/profile", User.UpdateProfile)
			auth.PUT("/user/password", User.ChangePassword)
			auth.GET("/user/info", User.GetUserInfo)
			auth.POST("/user/avatar", User.UploadAvatar)
			auth.GET("/avatar", User.GetAvatar)
			auth.GET("/avatar/by-uid", User.GetAvatarByUid)
			auth.POST("/file/upload", User.UploadFile)
			auth.POST("/message/recall", User.RecallMessage)
			auth.GET("/message/history", User.GetChatHistory)
			auth.POST("/moment/publish", User.PublishMoment)
			auth.GET("/moment/timeline", User.GetTimeline)
			auth.POST("/moment/like", User.LikeMoment)
			auth.POST("/moment/comment", User.CommentMoment)
			auth.POST("/moment/delete", User.DeleteMoment)
			auth.GET("/friend/list", User.GetFriends)
			auth.POST("/friend/request", User.AddFriend)
			auth.GET("/friend/requests", User.GetFriendRequests)
			auth.POST("/friend/accept", User.AcceptFriend)
			auth.POST("/friend/remove", User.RemoveFriend)
			auth.POST("/friend/remark", User.UpdateFriendRemark)
			auth.POST("/friend/block", User.BlockFriend)
			auth.POST("/friend/unblock", User.UnblockFriend)
			auth.GET("/friend/blocklist", User.GetBlockedList)
			auth.GET("/conversation/list", User.GetConversations)
			auth.POST("/group/create", User.CreateGroup)
			auth.GET("/group/list", User.GetMyGroups)
			auth.GET("/group/info", User.GetGroupInfo)
			auth.GET("/group/members", User.GetGroupMembers)
			auth.POST("/group/join", User.JoinGroup)
			auth.POST("/group/invite", User.InviteToGroup)
			auth.GET("/group/requests", User.GroupJoinRequests)
			auth.POST("/group/approve", User.ApproveJoin)
			auth.POST("/group/reject", User.RejectJoin)
			auth.POST("/group/leave", User.LeaveGroup)
			auth.POST("/group/disband", User.DisbandGroup)
			auth.POST("/group/kick", User.KickGroupMember)
			auth.POST("/group/transfer", User.TransferGroup)
			auth.POST("/group/mute", User.MuteGroupMember)
			auth.POST("/group/announce", User.SetGroupAnnouncement)
		}
	}

	if err := router.Run(fmt.Sprintf("%s:%d", s.address, s.port)); err != nil {
		panic(err)
	}
}
