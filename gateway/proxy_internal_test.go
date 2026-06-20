package gateway

import (
	"net"
	"testing"
	"time"
)

// listener 关闭后 serveProxy 必须及时退出，而不是 continue 忙等死循环。
func TestServeProxyExitsOnListenerClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	lb := NewLoadBalancer([]string{"127.0.0.1:0"})

	done := make(chan error, 1)
	go func() { done <- serveProxy(ln, lb) }()

	time.Sleep(50 * time.Millisecond)
	ln.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serveProxy returned error on listener close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveProxy did not exit after listener close (busy-loop bug)")
	}
}
