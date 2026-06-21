package http

import (
	"fmt"

	"IM/http/User"

	"github.com/gin-gonic/gin"
)

type Server struct {
	address string
	port    int
}

func NewServer(address string, port int) *Server {
	return &Server{
		address: address,
		port:    port,
	}
}
func (s *Server) Start() {
	router := gin.Default()

	api := router.Group("/api")
	{
		user := api.Group("/user")
		{
			user.POST("/register", User.Register)
			user.POST("/login", User.Login)
		}

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
