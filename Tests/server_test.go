package Tests

import (
	"IM/tcp"
	"IM/tcp/Message"
	"bytes"
	"net"
	"sync"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	s := tcp.NewServer("127.0.0.1", 9000, 10*time.Second)
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestServerGetConnectCountInitiallyZero(t *testing.T) {
	s := tcp.NewServer("127.0.0.1", 9000, 10*time.Second)
	count := s.GetConnectCount()
	if count != 0 {
		t.Errorf("initial connect count = %d, want 0", count)
	}
}

func TestServerAddHandler(t *testing.T) {
	s := tcp.NewServer("127.0.0.1", 9000, 10*time.Second)

	handler1 := tcp.Echo
	handler2 := tcp.Echo

	s.AddHandler(handler1)
	s.AddHandler(handler2)
}

func TestServerAddMultipleHandlers(t *testing.T) {
	s := tcp.NewServer("127.0.0.1", 9000, 10*time.Second)

	for i := 0; i < 10; i++ {
		s.AddHandler(tcp.Echo)
	}
}

func TestServerRegisterAndLookup(t *testing.T) {
	s := tcp.NewServer("127.0.0.1", 0, 10*time.Second)

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, s)
	s.Register("testuid", client)

	found, ok := s.Lookup("testuid")
	if !ok {
		t.Fatal("expected client to be registered")
	}
	if found != client {
		t.Error("lookup returned wrong client pointer")
	}
}

func TestServerLookupNotFound(t *testing.T) {
	s := tcp.NewServer("127.0.0.1", 0, 10*time.Second)

	_, ok := s.Lookup("noone")
	if ok {
		t.Error("expected lookup to return false")
	}
}

func TestServerRouteTo(t *testing.T) {
	s := tcp.NewServer("127.0.0.1", 0, 10*time.Second)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, s)
	s.Register("target", client)

	msg := Message.TextMessage(1, "hello")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.RouteTo("target", msg)
		if err != nil {
			t.Errorf("RouteTo failed: %v", err)
		}
	}()

	resp, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(resp.Data, []byte("hello")) {
		t.Errorf("received = %v, want 'hello'", resp.Data)
	}
}

func TestServerRouteToNotFound(t *testing.T) {
	s := tcp.NewServer("127.0.0.1", 0, 10*time.Second)

	msg := Message.TextMessage(1, "hello")
	err := s.RouteTo("missing", msg)
	if err == nil {
		t.Error("expected error for route to missing client")
	}
}

func TestServerShutDown(t *testing.T) {
	s := tcp.NewServer("127.0.0.1", 0, 10*time.Second)

	s.ShutDown()
	// should not panic
}
