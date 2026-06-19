package tcp

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

type Server struct {
	address        string
	port           int
	count          atomic.Int32
	t              time.Duration
	pool           *TieredPool
	clientHandlers []Handler
}

func NewServer(address string, port int) *Server {
	return &Server{
		address: address,
		port:    port,
		pool:    NewTieredPool(64, 256, 1024, 1024*4, 1024*16, 1024*64),
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
		s.count.Add(1)
		go NewClient(con).Start()
	}
}
func (s *Server) GetConnectCount() int32 {
	return s.count.Load()
}

func (s *Server) ShutDown() {

}

// AddHandler
func (s *Server) AddHandler(h Handler) {
	s.clientHandlers = append(s.clientHandlers, h)
}
