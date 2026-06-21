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

// 单聊“正在输入”：Json 帧(action=typing) 应转发给对端，且不触发离线同步。
func TestTypingSingleChatRoutesToPeer(t *testing.T) {
	server := NewServer("", 0, 10*time.Second)
	server.AddHandler(Router)

	senderConn, senderServerConn := net.Pipe()
	defer senderConn.Close()
	defer senderServerConn.Close()
	sender := NewClient(senderServerConn, server)
	sender.setUID("sender")
	go sender.MessageHandler()

	peerConn, peerServerConn := net.Pipe()
	defer peerConn.Close()
	defer peerServerConn.Close()
	peer := NewClient(peerServerConn, server)
	peer.setUID("peer")
	server.Register("peer", peer)

	body, _ := json.Marshal(map[string]string{"action": "typing", "to_uid": "peer"})
	sender.Process(Message.NewMessage(Message.Json, 1, body))

	f, err := readFrame(peerConn)
	if err != nil {
		t.Fatalf("read typing frame: %v", err)
	}
	if f.GetMsgType() != Message.Json {
		t.Fatalf("expected Json typing, got %d", f.GetMsgType())
	}
	var n map[string]any
	if err := json.Unmarshal(f.Data, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n["event"] != "typing" || n["from_uid"] != "sender" {
		t.Errorf("unexpected typing payload: %+v", n)
	}
}

// 群“正在输入”：Json 帧(action=typing, group_id) 应转发给除发送者外的在线成员。
func TestTypingGroupRoutesToMembers(t *testing.T) {
	origMembers := getGroupMembers
	defer func() { getGroupMembers = origMembers }()
	getGroupMembers = func(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
		return []*model.GroupMember{{Uid: "sender"}, {Uid: "m2"}}, nil
	}

	server := NewServer("", 0, 10*time.Second)
	server.AddHandler(Router)

	senderConn, senderServerConn := net.Pipe()
	defer senderConn.Close()
	defer senderServerConn.Close()
	sender := NewClient(senderServerConn, server)
	sender.setUID("sender")
	go sender.MessageHandler()

	m2Conn, m2ServerConn := net.Pipe()
	defer m2Conn.Close()
	defer m2ServerConn.Close()
	m2 := NewClient(m2ServerConn, server)
	m2.setUID("m2")
	server.Register("m2", m2)

	body, _ := json.Marshal(map[string]string{"action": "typing", "group_id": "g1"})
	sender.Process(Message.NewMessage(Message.Json, 1, body))

	f, err := readFrame(m2Conn)
	if err != nil {
		t.Fatalf("read group typing: %v", err)
	}
	var n map[string]any
	if err := json.Unmarshal(f.Data, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n["event"] != "typing" || n["from_uid"] != "sender" || n["group_id"] != "g1" {
		t.Errorf("unexpected group typing payload: %+v", n)
	}
}
