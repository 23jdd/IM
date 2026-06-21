package tcp

import (
	"IM/tcp/Message"
	"bytes"
	"net"
	"testing"
	"time"
)

// 同账号多端在线：RouteTo 应投递给该 uid 的所有连接。
func TestRouteToAllDevices(t *testing.T) {
	server := NewServer("", 0, 10*time.Second)

	d1Conn, d1Srv := net.Pipe()
	defer d1Conn.Close()
	defer d1Srv.Close()
	d2Conn, d2Srv := net.Pipe()
	defer d2Conn.Close()
	defer d2Srv.Close()

	c1 := NewClient(d1Srv, server)
	c1.setUID("bob")
	server.Register("bob", c1)
	c2 := NewClient(d2Srv, server)
	c2.setUID("bob")
	server.Register("bob", c2)

	res := make(chan []byte, 2)
	go func() {
		if f, err := readFrame(d1Conn); err == nil {
			res <- f.Data
		}
	}()
	go func() {
		if f, err := readFrame(d2Conn); err == nil {
			res <- f.Data
		}
	}()
	time.Sleep(50 * time.Millisecond) // 让两端先阻塞在读上

	if err := server.RouteTo("bob", Message.TextMessage(5, "hi")); err != nil {
		t.Fatalf("RouteTo: %v", err)
	}

	for i := 0; i < 2; i++ {
		select {
		case d := <-res:
			if !bytes.Equal(d, []byte("hi")) {
				t.Errorf("device got %q, want hi", d)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("a device did not receive the message")
		}
	}
}

// 多端同步：RouteToOthers 投递给同 uid 的其他连接，但不回发给发送连接。
func TestRouteToOthersExcludesSender(t *testing.T) {
	server := NewServer("", 0, 10*time.Second)

	c1Conn, c1Srv := net.Pipe()
	defer c1Conn.Close()
	defer c1Srv.Close()
	c2Conn, c2Srv := net.Pipe()
	defer c2Conn.Close()
	defer c2Srv.Close()

	c1 := NewClient(c1Srv, server)
	c1.setUID("alice")
	server.Register("alice", c1)
	c2 := NewClient(c2Srv, server)
	c2.setUID("alice")
	server.Register("alice", c2)

	res := make(chan []byte, 1)
	go func() {
		if f, err := readFrame(c2Conn); err == nil {
			res <- f.Data
		}
	}()
	time.Sleep(50 * time.Millisecond)

	go server.RouteToOthers("alice", c1, Message.TextMessage(9, "sync"))

	select {
	case d := <-res:
		if !bytes.Equal(d, []byte("sync")) {
			t.Errorf("other device got %q, want sync", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("other device did not receive the sync frame")
	}

	// 发送连接不应收到自己的同步帧。
	_ = c1Conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, err := readFrame(c1Conn); err == nil {
		t.Error("sender connection must not receive its own sync frame")
	}
}

// 引用计数：多端在线时，关闭其中一个连接不应使整个 uid 离线。
func TestRemoveClientRefcount(t *testing.T) {
	server := NewServer("", 0, 10*time.Second)
	_, s1 := net.Pipe()
	_, s2 := net.Pipe()
	defer s1.Close()
	defer s2.Close()

	c1 := NewClient(s1, server)
	c2 := NewClient(s2, server)
	server.Register("u", c1)
	server.Register("u", c2)

	if server.removeClient("u", c1) {
		t.Error("removing one of two connections should not be empty")
	}
	if _, ok := server.Lookup("u"); !ok {
		t.Error("uid should still be online while another connection remains")
	}
	if !server.removeClient("u", c2) {
		t.Error("removing the last connection should report empty")
	}
	if _, ok := server.Lookup("u"); ok {
		t.Error("uid should be offline after the last connection is removed")
	}
}
