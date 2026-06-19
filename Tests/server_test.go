package Tests

import (
	"IM/tcp"
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
