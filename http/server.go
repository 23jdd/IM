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
			auth.GET("/friend/list", User.GetFriends)
			auth.GET("/conversation/list", User.GetConversations)
		}
	}

	if err := router.Run(fmt.Sprintf("%s:%d", s.address, s.port)); err != nil {
		panic(err)
	}
}