package Tests

import (
	"IM/tcp"
	"IM/tcp/Message"
	"IM/utils"
	"bytes"
	"net"
	"sync"
	"testing"
	"time"
)

func TestVerifySuccess(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Verify)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	token, err := utils.GenerateToken("user001", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	msg := Message.AuthMessage(1, token)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.Process(msg)
	}()

	resp, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read response failed: %v", err)
	}
	wg.Wait()

	if resp.GetMsgType() != Message.ACK {
		t.Errorf("expected ACK, got type %d", resp.GetMsgType())
	}
	if resp.GetKey() != 1 {
		t.Errorf("expected key 1, got %d", resp.GetKey())
	}
	if client.UID() != "user001" {
		t.Errorf("expected uid 'user001', got '%s'", client.UID())
	}

	val, ok := server.Lookup("user001")
	if !ok {
		t.Fatal("client not registered in server.clients")
	}
	if val != client {
		t.Error("registered client pointer mismatch")
	}
}

func TestVerifyInvalidToken(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Verify)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	msg := Message.AuthMessage(2, "invalid-token")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.Process(msg)
	}()

	resp, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read response failed: %v", err)
	}
	wg.Wait()

	if resp.GetMsgType() != Message.Nack {
		t.Errorf("expected Nack, got type %d", resp.GetMsgType())
	}
	if resp.GetKey() != 2 {
		t.Errorf("expected key 2, got %d", resp.GetKey())
	}
	if client.UID() != "" {
		t.Errorf("expected empty uid after failed auth, got '%s'", client.UID())
	}
}

func TestVerifyReauth(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Verify)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	token, _ := utils.GenerateToken("user002", time.Now().Add(time.Hour))
	msg := Message.AuthMessage(1, token)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); client.Process(msg) }()
	readFullMessage(clientConn)
	wg.Wait()

	wg.Add(1)
	go func() { defer wg.Done(); client.Process(msg) }()
	resp, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read re-auth response failed: %v", err)
	}
	wg.Wait()

	if resp.GetMsgType() != Message.Nack {
		t.Errorf("expected Nack on re-auth, got type %d", resp.GetMsgType())
	}
}

func TestVerifyNonAuthIgnored(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Verify)
	server.AddHandler(tcp.Echo)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	msg := Message.TextMessage(5, "hello")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); client.Process(msg) }()

	resp, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read response failed: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(resp.Data, []byte("hello")) {
		t.Errorf("expected echo data 'hello', got %v", resp.Data)
	}
}

func TestRouteToSuccess(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)

	targetConn, serverConn := net.Pipe()
	defer targetConn.Close()
	defer serverConn.Close()

	target := tcp.NewClient(serverConn, server)
	go target.MessageHandler()

	server.Register("target001", target)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		msg := Message.TextMessage(42, "routed message")
		err := server.RouteTo("target001", msg)
		if err != nil {
			t.Errorf("RouteTo failed: %v", err)
		}
	}()

	resp, err := readFullMessage(targetConn)
	if err != nil {
		t.Fatalf("read routed message failed: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(resp.Data, []byte("routed message")) {
		t.Errorf("routed data = %v, want 'routed message'", resp.Data)
	}
}

func TestRouteToOffline(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)

	msg := Message.TextMessage(1, "nobody")
	err := server.RouteTo("offline_user", msg)
	if err == nil {
		t.Error("expected error for offline client")
	}
}
