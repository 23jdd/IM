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

type Backend struct {
	Addr   string
	Weight int
	Alive  bool
}

type LoadBalancer struct {
	mu       sync.RWMutex
	backends []*Backend
}

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

func (lb *LoadBalancer) AddBackend(addr string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.backends = append(lb.backends, &Backend{
		Addr:   addr,
		Weight: 1,
		Alive:  true,
	})
}

func StartTCPProxy(listenAddr string, lb *LoadBalancer) error {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	return serveProxy(listener, lb)
}

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
			clientConn.Close()
			continue
		}

		go func() {
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
