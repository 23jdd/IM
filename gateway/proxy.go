package gateway

import (
	"fmt"
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

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			continue
		}

		backend, err := lb.Pick()
		if err != nil {
			clientConn.Close()
			continue
		}

		go func() {
			defer clientConn.Close()

			backendConn, err := net.DialTimeout("tcp", backend.Addr, 5*time.Second)
			if err != nil {
				return
			}
			defer backendConn.Close()

			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				buf := make([]byte, 4096)
				for {
					n, err := clientConn.Read(buf)
					if err != nil {
						return
					}
					if _, err := backendConn.Write(buf[:n]); err != nil {
						return
					}
				}
			}()

			go func() {
				defer wg.Done()
				buf := make([]byte, 4096)
				for {
					n, err := backendConn.Read(buf)
					if err != nil {
						return
					}
					if _, err := clientConn.Write(buf[:n]); err != nil {
						return
					}
				}
			}()

			wg.Wait()
		}()
	}
}
