package tcp

import (
	"fmt"
	"log"
	"net"
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
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.address, s.port))
	if err != nil {
		panic(err)
	}
	for {
		con, err := listen.Accept()
		if err != nil {
			continue
		}
		log.Println(con.RemoteAddr().String())
		//TODO use ants
		go NewClient(con).Start()
	}
}
