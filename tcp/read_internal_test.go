package tcp

import (
	"IM/model"
	"IM/tcp/Message"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// 单聊“已读”回执：Json 帧(action=read) 应回给对端，带 up_to。
func TestReadReceiptSingleChatRoutesToPeer(t *testing.T) {
	server := NewServer("", 0, 10*time.Second)
	server.AddHandler(Router)

	senderConn, senderServerConn := net.Pipe()
	defer senderConn.Close()
	defer senderServerConn.Close()
	sender := NewClient(senderServerConn, server)
	sender.setUID("reader")
	go sender.MessageHandler()

	peerConn, peerServerConn := net.Pipe()
	defer peerConn.Close()
	defer peerServerConn.Close()
	peer := NewClient(peerServerConn, server)
	peer.setUID("author")
	server.Register("author", peer)

	body, _ := json.Marshal(map[string]any{"action": "read", "to_uid": "author", "up_to": 1234})
	sender.Process(Message.NewMessage(Message.Json, 1, body))

	f, err := readFrame(peerConn)
	if err != nil {
		t.Fatalf("read receipt frame: %v", err)
	}
	if f.GetMsgType() != Message.Json {
		t.Fatalf("expected Json read, got %d", f.GetMsgType())
	}
	var n map[string]any
	if err := json.Unmarshal(f.Data, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n["event"] != "read" || n["from_uid"] != "reader" || n["up_to"] != float64(1234) {
		t.Errorf("unexpected read payload: %+v", n)
	}
}

// 群“已读”回执：Json 帧(action=read, group_id) 应扇出给除阅读者外的在线成员。
func TestReadReceiptGroupRoutesToMembers(t *testing.T) {
	origMembers := getGroupMembers
	defer func() { getGroupMembers = origMembers }()
	getGroupMembers = func(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
		return []*model.GroupMember{{Uid: "reader"}, {Uid: "author"}}, nil
	}

	server := NewServer("", 0, 10*time.Second)
	server.AddHandler(Router)

	senderConn, senderServerConn := net.Pipe()
	defer senderConn.Close()
	defer senderServerConn.Close()
	sender := NewClient(senderServerConn, server)
	sender.setUID("reader")
	go sender.MessageHandler()

	authorConn, authorServerConn := net.Pipe()
	defer authorConn.Close()
	defer authorServerConn.Close()
	author := NewClient(authorServerConn, server)
	author.setUID("author")
	server.Register("author", author)

	body, _ := json.Marshal(map[string]any{"action": "read", "group_id": "g1", "up_to": 99})
	sender.Process(Message.NewMessage(Message.Json, 1, body))

	f, err := readFrame(authorConn)
	if err != nil {
		t.Fatalf("read group receipt: %v", err)
	}
	var n map[string]any
	if err := json.Unmarshal(f.Data, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n["event"] != "group_read" || n["from_uid"] != "reader" ||
		n["group_id"] != "g1" || n["up_to"] != float64(99) {
		t.Errorf("unexpected group_read payload: %+v", n)
	}
}
