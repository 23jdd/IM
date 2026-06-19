package tcp

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/panjf2000/ants/v2"
)

type Server struct {
	address        string
	port           int
	count          atomic.Int32
	t              time.Duration
	bufPool        *TieredPool
	workerPool     *ants.Pool
	clientHandlers []Handler
}

func NewServer(address string, port int, t time.Duration) *Server {
	wp, err := ants.NewPool(ants.DefaultAntsPoolSize)
	if err != nil {
		panic(err)
	}
	return &Server{
		address:    address,
		port:       port,
		bufPool:    NewTieredPool(8, 64, 256, 1024, 1024*4, 1024*16, 1024*64),
		t:          t,
		workerPool: wp,
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
		s.count.Add(1)
		err = s.workerPool.Submit(func() {
			NewClient(con, s).Start()
		})
		if err != nil {
			log.Println("submit to worker pool failed:", err)
			con.Close()
			s.count.Add(-1)
		}
	}
}

func (s *Server) GetConnectCount() int32 {
	return s.count.Load()
}

func (s *Server) ShutDown() {
	s.workerPool.Release()

}
func Notify() {
	// 创建接收信号的 channel
	sigCh := make(chan os.Signal, 1)
	// 监听 SIGINT 和 SIGTERM
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

}

func (s *Server) AddHandler(h Handler) {
	s.clientHandlers = append(s.clientHandlers, h)
}
