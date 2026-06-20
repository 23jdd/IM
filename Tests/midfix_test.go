package Tests

import (
	"IM/rabbitmq"
	"IM/tcp"
	"IM/tcp/Message"
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"
)

// #4 rabbitmq 未初始化时 Publish 应返回错误而非 panic。

func TestPublishMessageWithoutInit(t *testing.T) {
	err := rabbitmq.PublishMessage(context.Background(), &rabbitmq.MessageEvent{MsgId: "x"})
	if err == nil {
		t.Fatal("expected error when rabbitmq not initialized")
	}
}

func TestPublishNotificationWithoutInit(t *testing.T) {
	err := rabbitmq.PublishNotification(context.Background(), &rabbitmq.NotificationEvent{Uid: "x"})
	if err == nil {
		t.Fatal("expected error when rabbitmq not initialized")
	}
}

// #6 实时帧必须携带发送者信息。

func TestBuildRealtimeTextIncludesSender(t *testing.T) {
	body := tcp.BuildRealtimeText("sender1", "msg123", "hello", time.Now())
	var p tcp.RealtimeTextPayload
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.FromUid != "sender1" {
		t.Errorf("from_uid = %q, want sender1", p.FromUid)
	}
	if p.MsgId != "msg123" {
		t.Errorf("msg_id = %q, want msg123", p.MsgId)
	}
	if p.Content != "hello" {
		t.Errorf("content = %q, want hello", p.Content)
	}
}

func TestRouteToDeliversFrameWithSender(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)

	targetConn, serverConn := net.Pipe()
	defer targetConn.Close()
	defer serverConn.Close()

	target := tcp.NewClient(serverConn, server)
	go target.MessageHandler()
	server.Register("target", target)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		body := tcp.BuildRealtimeText("alice", "m1", "hi bob", time.Now())
		if err := server.RouteTo("target", Message.NewMessage(Message.Text, 0, body)); err != nil {
			t.Errorf("RouteTo failed: %v", err)
		}
	}()

	resp, err := readFullMessage(targetConn)
	if err != nil {
		t.Fatalf("read delivered frame: %v", err)
	}
	wg.Wait()

	if resp.GetMsgType() != Message.Text {
		t.Errorf("delivered type = %d, want Text", resp.GetMsgType())
	}
	var p tcp.RealtimeTextPayload
	if err := json.Unmarshal(resp.Data, &p); err != nil {
		t.Fatalf("unmarshal delivered frame: %v", err)
	}
	if p.FromUid != "alice" {
		t.Errorf("delivered from_uid = %q, want alice", p.FromUid)
	}
	if p.Content != "hi bob" {
		t.Errorf("delivered content = %q, want 'hi bob'", p.Content)
	}
}
