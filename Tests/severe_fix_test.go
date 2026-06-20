package Tests

import (
	"IM/tcp"
	"IM/tcp/Message"
	"IM/utils"
	"bytes"
	"encoding/binary"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- 严重问题 #3：ReadMessage 必须拒绝超大包体，防止 OOM / 溢出 ----

func TestReadMessageRejectsOversizedBody(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)

	go func() {
		header := make([]byte, 8)
		header[0] = Message.Text
		binary.BigEndian.PutUint32(header[4:8], uint32(tcp.MaxBodyLen)+1)
		_, _ = clientConn.Write(header)
	}()

	_, err := client.ReadMessage()
	if err == nil {
		t.Fatal("expected error for oversized body, got nil")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' error, got: %v", err)
	}
}

func TestReadMessageAcceptsValidBody(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)

	want := Message.TextMessage(7, "hello world")
	go func() { _, _ = clientConn.Write(Message.Encode(want)) }()

	got, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GetMsgType() != Message.Text || got.GetKey() != 7 ||
		!bytes.Equal(got.Data, []byte("hello world")) {
		t.Errorf("roundtrip mismatch: type=%d key=%d data=%q",
			got.GetMsgType(), got.GetKey(), got.Data)
	}
}

func TestReadMessageBoundaryRejectedAtLimitPlusOne(t *testing.T) {
	if tcp.MaxBodyLen <= 0 {
		t.Fatalf("MaxBodyLen must be positive, got %d", tcp.MaxBodyLen)
	}
}

// ---- 严重问题 #2：心跳 / 关闭必须能干净退出 goroutine，不泄漏 ----

func TestHeartBeatExitsOnClose(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Millisecond)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	client := tcp.NewClient(serverConn, server)

	done := make(chan struct{})
	go func() {
		client.HeartBeat()
		close(done)
	}()

	time.Sleep(30 * time.Millisecond) // 让 ticker 触发几次
	client.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("HeartBeat goroutine did not exit after Close (goroutine leak)")
	}
}

func TestStartCleansUpOnPeerClose(t *testing.T) {
	server := tcp.NewServer("", 0, 20*time.Millisecond)
	clientConn, serverConn := net.Pipe()

	client := tcp.NewClient(serverConn, server)

	// 持续读取，避免服务端心跳写入阻塞。
	go func() {
		buf := make([]byte, 256)
		for {
			if _, err := clientConn.Read(buf); err != nil {
				return
			}
		}
	}()

	go client.Start()
	time.Sleep(40 * time.Millisecond) // 让读循环 / 心跳跑起来

	clientConn.Close() // 对端断开 -> 读循环出错 -> 触发清理

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if client.IsClosed() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("client did not clean up after peer close")
}

// ---- 严重问题 #1：closed / uid 字段的并发安全（用 -race 运行验证）----

func TestSendReturnsErrorAfterClose(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	client := tcp.NewClient(serverConn, server)
	client.Close()

	if !client.IsClosed() {
		t.Fatal("expected IsClosed() true after Close")
	}
	if err := client.Send(Message.TextMessage(1, "x")); err == nil {
		t.Fatal("expected error when sending after close")
	}
}

func TestClientConcurrentClose(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	client := tcp.NewClient(serverConn, server)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.Close()
			_ = client.UID()
			_ = client.IsClosed()
			_ = client.Send(Message.HeartMessage(1))
		}()
	}
	wg.Wait()

	if !client.IsClosed() {
		t.Fatal("client should be closed")
	}
}

// 认证写 uid 与关闭读 uid 并发：旧实现存在数据竞争，加锁后应通过 -race。
func TestClientAuthCloseRace(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Verify)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	token, err := utils.GenerateToken("raceuser", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// 排空服务端写出的响应，避免 Send 阻塞。
	go func() {
		for {
			if _, err := readFullMessage(clientConn); err != nil {
				return
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		client.Process(Message.AuthMessage(1, token)) // -> Verify -> setUID
	}()
	go func() {
		defer wg.Done()
		client.Close() // -> 读取 uid
	}()
	wg.Wait()

	_ = client.UID()
}
