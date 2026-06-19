package http

import (
	"fmt"

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
	//gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	if err := router.Run(fmt.Sprintf("%s:%d", s.address, s.port)); err != nil {
		panic(err)
	}
}
