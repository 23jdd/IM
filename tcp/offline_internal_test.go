package tcp

import (
	"IM/model"
	"IM/tcp/Message"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func readFrame(conn net.Conn) (*Message.Message, error) {
	head := make([]byte, 8)
	if _, err := io.ReadFull(conn, head); err != nil {
		return nil, err
	}
	blen := binary.BigEndian.Uint32(head[4:8])
	buf := make([]byte, 8+blen)
	copy(buf, head)
	if _, err := io.ReadFull(conn, buf[8:]); err != nil {
		return nil, err
	}
	return Message.Decode(buf)
}

// P1：离线消息必须在客户端逐条 ACK 后才标记已读（at-least-once 可靠投递）。
func TestOfflineSyncMarksReadOnlyAfterAck(t *testing.T) {
	origGet := getOfflineMessages
	origMark := markMessagesRead
	defer func() { getOfflineMessages = origGet; markMessagesRead = origMark }()

	now := time.Now()
	getOfflineMessages = func(ctx context.Context, uid string) ([]*model.ChatMessage, error) {
		return []*model.ChatMessage{
			{MsgId: "m1", FromUid: "a", ToUid: uid, Content: "hi1", CreatedAt: now},
			{MsgId: "m2", FromUid: "a", ToUid: uid, Content: "hi2", CreatedAt: now},
		}, nil
	}
	markedCh := make(chan string, 8)
	markMessagesRead = func(ctx context.Context, ids []string) error {
		for _, id := range ids {
			markedCh <- id
		}
		return nil
	}

	server := NewServer("", 0, 10*time.Second)
	server.AddHandler(Router)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := NewClient(serverConn, server)
	client.setUID("bob") // 模拟已认证
	go client.MessageHandler()

	// 触发离线同步
	client.Process(Message.NewMessage(Message.Json, 1, nil))

	// 读取两条离线 blob（带非 0 key）
	blob1, err := readFrame(clientConn)
	if err != nil {
		t.Fatalf("read blob1: %v", err)
	}
	blob2, err := readFrame(clientConn)
	if err != nil {
		t.Fatalf("read blob2: %v", err)
	}
	if blob1.GetMsgType() != Message.Blob || blob2.GetMsgType() != Message.Blob {
		t.Fatalf("expected Blob frames, got %d / %d", blob1.GetMsgType(), blob2.GetMsgType())
	}
	if blob1.GetKey() == 0 || blob2.GetKey() == 0 {
		t.Fatal("offline blobs must carry non-zero key for ACK")
	}

	// 尚未 ACK：不应有任何标记已读
	select {
	case id := <-markedCh:
		t.Fatalf("message %s marked read before ACK", id)
	default:
	}

	// 客户端逐条 ACK
	client.Process(Message.AckMessage(blob1.GetKey()))
	client.Process(Message.AckMessage(blob2.GetKey()))

	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case id := <-markedCh:
			got[id] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for mark-read after ACK (got %v)", got)
		}
	}
	if !got["m1"] || !got["m2"] {
		t.Errorf("expected m1 and m2 marked read, got %v", got)
	}
}

func TestAckUnknownKeyIsNoop(t *testing.T) {
	origMark := markMessagesRead
	defer func() { markMessagesRead = origMark }()

	called := make(chan struct{}, 1)
	markMessagesRead = func(ctx context.Context, ids []string) error {
		called <- struct{}{}
		return nil
	}

	server := NewServer("", 0, 10*time.Second)
	server.AddHandler(Router)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := NewClient(serverConn, server)
	client.setUID("bob")
	go client.MessageHandler()

	// 发送一个未被跟踪的 ACK key
	client.Process(Message.AckMessage(9999))

	select {
	case <-called:
		t.Fatal("markMessagesRead should not be called for unknown ACK key")
	case <-time.After(200 * time.Millisecond):
		// 预期：无调用
	}
}
