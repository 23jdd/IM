package tcp

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"IM/tcp/Message"

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

	// 跨实例路由（P3）：单机时可为空，行为退化为本地路由。
	instanceID string
	presence   Presence
	forwarder  Forwarder
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

func (s *Server) RouteTo(uid string, m *Message.Message) error {
	// 1) 目标在本实例：直接投递。
	if val, ok := s.clients.Load(uid); ok {
		if c, ok := val.(*Client); ok {
			return c.Send(m)
		}
	}

	// 2) 目标在其他实例：经在线表查到实例并跨实例转发。
	if s.presence != nil && s.forwarder != nil {
		inst, err := s.presence.GetInstance(context.Background(), uid)
		if err == nil && inst != "" && inst != s.instanceID {
			return s.forwarder.Forward(context.Background(), inst, uid, Message.Encode(m))
		}
	}

	return fmt.Errorf("client %s not online", uid)
}

// DeliverLocal 将一条已编码的帧投递给本实例上的目标客户端，
// 由跨实例转发的接收端（订阅循环）调用。
func (s *Server) DeliverLocal(uid string, frame []byte) error {
	val, ok := s.clients.Load(uid)
	if !ok {
		return fmt.Errorf("client %s not on this instance", uid)
	}
	c, ok := val.(*Client)
	if !ok {
		return fmt.Errorf("invalid client type for %s", uid)
	}
	msg, err := Message.Decode(frame)
	if err != nil {
		return err
	}
	return c.Send(msg)
}

// SetInstanceID 设置本实例标识（用于跨实例路由区分自身）。
func (s *Server) SetInstanceID(id string) { s.instanceID = id }

// InstanceID 返回本实例标识。
func (s *Server) InstanceID() string { return s.instanceID }

// SetPresence 注入在线注册表。
func (s *Server) SetPresence(p Presence) { s.presence = p }

// SetForwarder 注入跨实例转发器。
func (s *Server) SetForwarder(f Forwarder) { s.forwarder = f }

func (s *Server) Register(uid string, c *Client) {
	s.clients.Store(uid, c)
}

func (s *Server) Lookup(uid string) (*Client, bool) {
	val, ok := s.clients.Load(uid)
	if !ok {
		return nil, false
	}
	c, ok := val.(*Client)
	return c, ok
}

func NotifyServer(s *Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	s.ShutDown()
}