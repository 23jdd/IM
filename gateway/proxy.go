package gateway

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Backend 表示一个后端服务节点。
type Backend struct {
	Addr   string // 后端地址
	Weight int    // 权重（暂未参与选择算法）
	Alive  bool   // 是否存活
}

// LoadBalancer 负载均衡器，维护后端列表并提供选择能力。
type LoadBalancer struct {
	mu       sync.RWMutex // 保护 backends 的读写锁
	backends []*Backend   // 后端节点列表
}

// NewLoadBalancer 根据地址列表创建负载均衡器，并启动后台健康检查协程。
func NewLoadBalancer(addrs []string) *LoadBalancer {
	lb := &LoadBalancer{}
	for _, addr := range addrs {
		lb.backends = append(lb.backends, &Backend{
			Addr:   addr,
			Weight: 1,
			Alive:  true,
		})
	}
	go lb.healthCheck()
	return lb
}

// healthCheck 每 5 秒探测一次所有后端的 TCP 可达性，更新其存活状态。
func (lb *LoadBalancer) healthCheck() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		lb.mu.Lock()
		for _, b := range lb.backends {
			conn, err := net.DialTimeout("tcp", b.Addr, 2*time.Second)
			if err != nil {
				b.Alive = false
			} else {
				b.Alive = true
				conn.Close()
			}
		}
		lb.mu.Unlock()
	}
}

// Pick 从存活的后端中随机挑选一个返回；无存活后端时返回错误。
func (lb *LoadBalancer) Pick() (*Backend, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var alive []*Backend
	for _, b := range lb.backends {
		if b.Alive {
			alive = append(alive, b)
		}
	}

	if len(alive) == 0 {
		return nil, fmt.Errorf("no alive backend")
	}

	return alive[rand.Intn(len(alive))], nil
}

// AddBackend 动态新增一个后端节点。
func (lb *LoadBalancer) AddBackend(addr string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.backends = append(lb.backends, &Backend{
		Addr:   addr,
		Weight: 1,
		Alive:  true,
	})
}

// StartTCPProxy 在指定地址监听并启动 TCP 代理服务。
func StartTCPProxy(listenAddr string, lb *LoadBalancer) error {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	return serveProxy(listener, lb)
}

// serveProxy 循环接受客户端连接，为每个连接选择后端并启动转发协程。
func serveProxy(listener net.Listener, lb *LoadBalancer) error {
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			// listener 关闭或致命错误时退出，避免原先 continue 导致的 CPU 忙等死循环。
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		backend, err := lb.Pick()
		if err != nil {
			clientConn.Close() // 无可用后端，关闭连接并继续接受下一个
			continue
		}

		go func() {
			// 拨号后端，失败则关闭客户端连接
			backendConn, derr := net.DialTimeout("tcp", backend.Addr, 5*time.Second)
			if derr != nil {
				clientConn.Close()
				return
			}
			ProxyConn(clientConn, backendConn)
		}()
	}
}

// ProxyConn 在两个连接间双向转发数据。
// 当任一方向结束（EOF 或出错）时，立即关闭两个连接，
// 使另一方向的拷贝也及时退出，避免连接 / 协程长时间残留（修复半关闭悬挂问题）。
func ProxyConn(a, b net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(b, a)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(a, b)
		done <- struct{}{}
	}()

	<-done // 任一方向结束
	a.Close()
	b.Close()
	<-done // 等待另一方向退出，确认无协程泄漏
}
