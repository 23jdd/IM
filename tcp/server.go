package tcp

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
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
	clients        sync.Map
	listener       net.Listener
	quit           chan struct{}
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
		quit:       make(chan struct{}),
	}
}

func (s *Server) Start() {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.address, s.port))
	if err != nil {
		panic(err)
	}
	s.listener = listen

	for {
		con, err := listen.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				continue
			}
		}
		log.Println("new connection:", con.RemoteAddr().String())
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
	log.Println("server shutting down...")
	close(s.quit)

	if s.listener != nil {
		s.listener.Close()
	}

	s.clients.Range(func(key, value any) bool {
		if c, ok := value.(*Client); ok {
			c.Close()
		}
		return true
	})

	s.workerPool.Release()
	log.Println("server stopped")
}

func (s *Server) AddHandler(h Handler) {
	s.clientHandlers = append(s.clientHandlers, h)
}

func NotifyServer(s *Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	s.ShutDown()
}