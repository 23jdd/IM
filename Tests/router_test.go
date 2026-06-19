package Tests

import (
	"IM/tcp"
	"IM/tcp/Message"
	"IM/utils"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"
)

func TestRouterUnauthenticated(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Router)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	msg := Message.TextMessage(1, "unauth")
	client.Process(msg)

	// No response expected — Router skips unauthenticated
	// If it didn't skip, Echo would respond (but Echo not registered here)
	// Just verify no panic/timeout
}

func TestRouterDispatch(t *testing.T) {
	received := make(chan *Message.Message, 1)

	tcp.RegisterRoute(100, func(m *Message.Message, c *tcp.Client) {
		received <- m
	})

	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Verify)
	server.AddHandler(tcp.Router)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	token, _ := utils.GenerateToken("user_router", time.Now().Add(time.Hour))
	authMsg := Message.AuthMessage(1, token)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); client.Process(authMsg) }()
	readFullMessage(clientConn)
	wg.Wait()

	msg := Message.NewMessage(100, 42, []byte("test_data"))
	client.Process(msg)

	select {
	case m := <-received:
		if m.GetMsgType() != 100 {
			t.Errorf("expected type 100, got %d", m.GetMsgType())
		}
		if m.GetKey() != 42 {
			t.Errorf("expected key 42, got %d", m.GetKey())
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for routed message")
	}
}

func TestRouterUnknownType(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Verify)
	server.AddHandler(tcp.Router)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	token, _ := utils.GenerateToken("user_unk", time.Now().Add(time.Hour))
	authMsg := Message.AuthMessage(1, token)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); client.Process(authMsg) }()
	readFullMessage(clientConn)
	wg.Wait()

	msg := Message.NewMessage(99, 1, nil)
	client.Process(msg)
	// should not panic, just log unknown type
}

func TestChatMessagePayload(t *testing.T) {
	payload := tcp.TextChatPayload{
		ToUid:   "target_user",
		Content: "hello world",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded tcp.TextChatPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ToUid != "target_user" {
		t.Errorf("ToUid = %s, want target_user", decoded.ToUid)
	}
	if decoded.Content != "hello world" {
		t.Errorf("Content = %s, want hello world", decoded.Content)
	}
}

func TestRouterChainWithFallback(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Verify)
	server.AddHandler(tcp.Router)
	server.AddHandler(tcp.Echo)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	token, _ := utils.GenerateToken("user_chain", time.Now().Add(time.Hour))
	authMsg := Message.AuthMessage(1, token)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); client.Process(authMsg) }()
	readFullMessage(clientConn)
	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		msg := Message.TextMessage(5, "echo_fallback")
		client.Process(msg)
	}()

	resp, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	wg.Wait()

	if string(resp.Data) != "echo_fallback" {
		t.Errorf("echo data = %s, want echo_fallback", resp.Data)
	}
}
