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

// 群聊：消息持久化后扇出给所有在线群成员（跳过发送者本人）。
func TestGroupMessageFanOut(t *testing.T) {
	origMembers := getGroupMembers
	origSend := sendGroupMessage
	defer func() { getGroupMembers = origMembers; sendGroupMessage = origSend }()

	getGroupMembers = func(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
		return []*model.GroupMember{
			{Uid: "sender"}, // 应被跳过
			{Uid: "m2"},
			{Uid: "m3"},
		}, nil
	}
	now := time.Now()
	sendGroupMessage = func(ctx context.Context, fromUid, groupId string, msgType byte, content string) (*model.ChatMessage, error) {
		return &model.ChatMessage{
			MsgId:     "gm1",
			FromUid:   fromUid,
			GroupId:   groupId,
			Content:   content,
			CreatedAt: now,
		}, nil
	}

	server := NewServer("", 0, 10*time.Second)
	server.AddHandler(Router)

	// 发送者
	senderConn, senderServerConn := net.Pipe()
	defer senderConn.Close()
	defer senderServerConn.Close()
	sender := NewClient(senderServerConn, server)
	sender.setUID("sender")
	go sender.MessageHandler()

	// 两个在线成员
	m2Conn, m2ServerConn := net.Pipe()
	defer m2Conn.Close()
	defer m2ServerConn.Close()
	m2 := NewClient(m2ServerConn, server)
	m2.setUID("m2")
	server.Register("m2", m2)

	m3Conn, m3ServerConn := net.Pipe()
	defer m3Conn.Close()
	defer m3ServerConn.Close()
	m3 := NewClient(m3ServerConn, server)
	m3.setUID("m3")
	server.Register("m3", m3)

	body, _ := json.Marshal(map[string]string{"group_id": "g1", "content": "hello group"})
	sender.Process(Message.NewMessage(Message.Text, 1, body))

	// 写顺序：先 ACK 给 sender，再扇出 m2、m3
	ack, err := readFrame(senderConn)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.GetMsgType() != Message.ACK || ack.GetKey() != 1 {
		t.Errorf("expected ACK key=1, got type=%d key=%d", ack.GetMsgType(), ack.GetKey())
	}

	check := func(conn net.Conn, who string) {
		f, err := readFrame(conn)
		if err != nil {
			t.Fatalf("%s read fanout frame: %v", who, err)
		}
		var p RealtimeTextPayload
		if err := json.Unmarshal(f.Data, &p); err != nil {
			t.Fatalf("%s unmarshal: %v", who, err)
		}
		if p.FromUid != "sender" || p.GroupId != "g1" || p.Content != "hello group" {
			t.Errorf("%s frame mismatch: %+v", who, p)
		}
	}
	check(m2Conn, "m2")
	check(m3Conn, "m3")
}
