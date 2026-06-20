package Tests

import (
	"IM/gateway"
	"io"
	"net"
	"testing"
	"time"
)

// #7 代理双向中继正常工作。
func TestProxyConnRelaysData(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	go func() {
		c, err := backend.Accept()
		if err != nil {
			return
		}
		_, _ = io.Copy(c, c) // echo
	}()

	proxyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer proxyLn.Close()

	go func() {
		clientSide, err := proxyLn.Accept()
		if err != nil {
			return
		}
		backendSide, err := net.Dial("tcp", backend.Addr().String())
		if err != nil {
			clientSide.Close()
			return
		}
		gateway.ProxyConn(clientSide, backendSide)
	}()

	client, err := net.Dial("tcp", proxyLn.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4)
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(buf) != "ping" {
		t.Errorf("echo = %q, want ping", buf)
	}
}

// #7 核心：一端关闭后，代理必须及时关闭另一端，不能悬挂。
func TestProxyConnClosesPeerWhenOneSideCloses(t *testing.T) {
	la, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer la.Close()
	lb, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lb.Close()

	aPeerCh := make(chan net.Conn, 1)
	bPeerCh := make(chan net.Conn, 1)
	go func() {
		if c, err := la.Accept(); err == nil {
			aPeerCh <- c
		}
	}()
	go func() {
		if c, err := lb.Accept(); err == nil {
			bPeerCh <- c
		}
	}()

	aSide, err := net.Dial("tcp", la.Addr().String()) // 代理的 "a" 连接（client 侧）
	if err != nil {
		t.Fatal(err)
	}
	bSide, err := net.Dial("tcp", lb.Addr().String()) // 代理的 "b" 连接（backend 侧）
	if err != nil {
		t.Fatal(err)
	}

	aPeer := <-aPeerCh // 模拟 client 端
	bPeer := <-bPeerCh // 模拟 backend 端
	defer bPeer.Close()

	go gateway.ProxyConn(aSide, bSide)

	// client 端关闭 -> aSide EOF -> ProxyConn 关闭 bSide -> bPeer 观察到连接关闭
	aPeer.Close()

	_ = bPeer.SetReadDeadline(time.Now().Add(2 * time.Second))
	start := time.Now()
	buf := make([]byte, 16)
	_, err = bPeer.Read(buf)
	if err == nil {
		t.Fatal("expected backend peer to observe closed connection")
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Fatal("backend peer read timed out -> proxy did NOT close the peer (half-close hang not fixed)")
	}
	if time.Since(start) > time.Second {
		t.Errorf("peer close propagation too slow: %v", time.Since(start))
	}
}
