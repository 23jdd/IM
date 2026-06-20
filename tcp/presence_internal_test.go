package tcp

import (
	"IM/tcp/Message"
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// 测试用的内存转发器：把帧投递到目标实例的 Server.DeliverLocal。
type testForwarder struct {
	targets map[string]*Server
	calls   int
}

func (f *testForwarder) Forward(ctx context.Context, instance, toUid string, frame []byte) error {
	f.calls++
	srv, ok := f.targets[instance]
	if !ok {
		return fmt.Errorf("no target instance %s", instance)
	}
	return srv.DeliverLocal(toUid, frame)
}

func TestMemoryPresence(t *testing.T) {
	p := NewMemoryPresence()
	ctx := context.Background()

	if inst, _ := p.GetInstance(ctx, "u1"); inst != "" {
		t.Errorf("expected empty instance for offline user, got %q", inst)
	}

	_ = p.SetOnline(ctx, "u1", "instA")
	if inst, _ := p.GetInstance(ctx, "u1"); inst != "instA" {
		t.Errorf("GetInstance = %q, want instA", inst)
	}

	// 用别的实例 SetOffline 不应删除（防误删用户在别处的新会话）
	_ = p.SetOffline(ctx, "u1", "instB")
	if inst, _ := p.GetInstance(ctx, "u1"); inst != "instA" {
		t.Errorf("SetOffline by wrong instance removed entry: %q", inst)
	}

	// 本实例 SetOffline 正常删除
	_ = p.SetOffline(ctx, "u1", "instA")
	if inst, _ := p.GetInstance(ctx, "u1"); inst != "" {
		t.Errorf("expected offline after correct SetOffline, got %q", inst)
	}
}

// P3 核心：目标用户在另一实例时，RouteTo 经在线表 + 转发器投递过去。
func TestRouteToCrossInstance(t *testing.T) {
	presence := NewMemoryPresence()

	s1 := NewServer("", 0, 10*time.Second)
	s1.SetInstanceID("inst1")
	s1.SetPresence(presence)

	s2 := NewServer("", 0, 10*time.Second)
	s2.SetInstanceID("inst2")
	s2.SetPresence(presence)

	fwd := &testForwarder{targets: map[string]*Server{"inst2": s2}}
	s1.SetForwarder(fwd)

	// bob 连接在 s2
	bobConn, bobServerConn := net.Pipe()
	defer bobConn.Close()
	defer bobServerConn.Close()
	bob := NewClient(bobServerConn, s2)
	bob.setUID("bob")
	s2.Register("bob", bob)
	_ = presence.SetOnline(context.Background(), "bob", "inst2")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s1.RouteTo("bob", Message.TextMessage(7, "cross hi")); err != nil {
			t.Errorf("cross-instance RouteTo failed: %v", err)
		}
	}()

	resp, err := readFrame(bobConn)
	if err != nil {
		t.Fatalf("read delivered frame: %v", err)
	}
	wg.Wait()

	if resp.GetMsgType() != Message.Text || resp.GetKey() != 7 ||
		!bytes.Equal(resp.Data, []byte("cross hi")) {
		t.Errorf("delivered frame mismatch: type=%d key=%d data=%q",
			resp.GetMsgType(), resp.GetKey(), resp.Data)
	}
	if fwd.calls != 1 {
		t.Errorf("forwarder calls = %d, want 1", fwd.calls)
	}
}

func TestRouteToLocalTakesPriority(t *testing.T) {
	presence := NewMemoryPresence()
	s1 := NewServer("", 0, 10*time.Second)
	s1.SetInstanceID("inst1")
	s1.SetPresence(presence)
	fwd := &testForwarder{targets: map[string]*Server{}}
	s1.SetForwarder(fwd)

	aliceConn, aliceServerConn := net.Pipe()
	defer aliceConn.Close()
	defer aliceServerConn.Close()
	alice := NewClient(aliceServerConn, s1)
	alice.setUID("alice")
	s1.Register("alice", alice)
	// 陈旧/错误的在线登记指向别的实例——但本地存在，应优先本地投递。
	_ = presence.SetOnline(context.Background(), "alice", "inst2")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s1.RouteTo("alice", Message.TextMessage(1, "local")); err != nil {
			t.Errorf("local RouteTo failed: %v", err)
		}
	}()

	resp, err := readFrame(aliceConn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(resp.Data, []byte("local")) {
		t.Errorf("data = %q, want local", resp.Data)
	}
	if fwd.calls != 0 {
		t.Errorf("forwarder should not be called for local delivery, calls=%d", fwd.calls)
	}
}

func TestRouteToOfflineReturnsError(t *testing.T) {
	presence := NewMemoryPresence()
	s1 := NewServer("", 0, 10*time.Second)
	s1.SetInstanceID("inst1")
	s1.SetPresence(presence)
	s1.SetForwarder(&testForwarder{targets: map[string]*Server{}})

	if err := s1.RouteTo("ghost", Message.TextMessage(1, "x")); err == nil {
		t.Fatal("expected error routing to offline user")
	}
}
