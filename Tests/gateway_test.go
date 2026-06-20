package Tests

import (
	"IM/gateway"
	"testing"
)

func TestNewLoadBalancer(t *testing.T) {
	addrs := []string{"127.0.0.1:9000", "127.0.0.1:9001", "127.0.0.1:9002"}
	lb := gateway.NewLoadBalancer(addrs)
	if lb == nil {
		t.Fatal("NewLoadBalancer returned nil")
	}
}

func TestLoadBalancerPick(t *testing.T) {
	lb := gateway.NewLoadBalancer([]string{"127.0.0.1:9000"})

	backend, err := lb.Pick()
	if err != nil {
		t.Fatalf("Pick failed: %v", err)
	}
	if backend.Addr != "127.0.0.1:9000" {
		t.Errorf("addr = %s, want 127.0.0.1:9000", backend.Addr)
	}
}

func TestLoadBalancerPickWhenAlive(t *testing.T) {
	lb := gateway.NewLoadBalancer([]string{"127.0.0.1:19999"})

	backend, err := lb.Pick()
	if err != nil {
		t.Fatalf("Pick failed: %v", err)
	}
	if backend.Addr != "127.0.0.1:19999" {
		t.Errorf("addr = %s, want 127.0.0.1:19999", backend.Addr)
	}
}

func TestLoadBalancerAddBackend(t *testing.T) {
	lb := gateway.NewLoadBalancer([]string{})
	lb.AddBackend("127.0.0.1:9000")
	lb.AddBackend("127.0.0.1:9001")

	backend, err := lb.Pick()
	if err != nil {
		t.Fatalf("Pick failed: %v", err)
	}
	if backend.Addr != "127.0.0.1:9000" && backend.Addr != "127.0.0.1:9001" {
		t.Errorf("unexpected backend: %s", backend.Addr)
	}
}
