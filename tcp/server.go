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

	// 先快照所有连接再逐个关闭：Close 内部会回调 removeClient 加写锁，
	// 不能在持有 connSet 读锁时调用，故收集后在锁外关闭。
	var all []*Client
	s.clients.Range(func(_, value any) bool {
		if cs, ok := value.(*connSet); ok {
			cs.mu.RLock()
			for c := range cs.m {
				all = append(all, c)
			}
			cs.mu.RUnlock()
		}
		return true
	})
	for _, c := range all {
		c.Close()
	}

	s.workerPool.Release()
	log.Println("server stopped")
}

func (s *Server) AddHandler(h Handler) {
	s.clientHandlers = append(s.clientHandlers, h)
}

func (s *Server) RouteTo(uid string, m *Message.Message) error {
	// 1) 目标在本实例：投递给其所有在线连接（多端在线）。
	if locals := s.localClients(uid); len(locals) > 0 {
		var firstErr error
		for _, c := range locals {
			if err := c.Send(m); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
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

// DeliverLocal 将一条已编码的帧投递给本实例上该 uid 的所有连接，
// 由跨实例转发的接收端（订阅循环）调用。
func (s *Server) DeliverLocal(uid string, frame []byte) error {
	locals := s.localClients(uid)
	if len(locals) == 0 {
		return fmt.Errorf("client %s not on this instance", uid)
	}
	msg, err := Message.Decode(frame)
	if err != nil {
		return err
	}
	var firstErr error
	for _, c := range locals {
		if err := c.Send(msg); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
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
	s.addClient(uid, c)
}

func (s *Server) Lookup(uid string) (*Client, bool) {
	locals := s.localClients(uid)
	if len(locals) == 0 {
		return nil, false
	}
	return locals[0], true
}

// connSet 持有同一 uid 的多条连接（多端在线），并发安全。
type connSet struct {
	mu sync.RWMutex
	m  map[*Client]struct{}
}

// addClient 把连接加入该 uid 的连接集合（支持同账号多端在线）。
func (s *Server) addClient(uid string, c *Client) {
	val, _ := s.clients.LoadOrStore(uid, &connSet{m: make(map[*Client]struct{})})
	cs := val.(*connSet)
	cs.mu.Lock()
	cs.m[c] = struct{}{}
	cs.mu.Unlock()
}

// removeClient 从该 uid 的集合移除连接；返回该 uid 是否已无任何在线连接。
func (s *Server) removeClient(uid string, c *Client) bool {
	val, ok := s.clients.Load(uid)
	if !ok {
		return true
	}
	cs := val.(*connSet)
	cs.mu.Lock()
	delete(cs.m, c)
	empty := len(cs.m) == 0
	cs.mu.Unlock()
	if empty {
		s.clients.Delete(uid)
	}
	return empty
}

// localClients 返回该 uid 在本实例上的所有连接快照。
func (s *Server) localClients(uid string) []*Client {
	val, ok := s.clients.Load(uid)
	if !ok {
		return nil
	}
	cs := val.(*connSet)
	cs.mu.RLock()
	out := make([]*Client, 0, len(cs.m))
	for c := range cs.m {
		out = append(out, c)
	}
	cs.mu.RUnlock()
	return out
}

// RouteToOthers 把帧投递给该 uid 在本实例上、除 except 外的其他连接（多端同步）。
func (s *Server) RouteToOthers(uid string, except *Client, m *Message.Message) {
	for _, c := range s.localClients(uid) {
		if c == except {
			continue
		}
		_ = c.Send(m)
	}
}

func NotifyServer(s *Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	s.ShutDown()
}
