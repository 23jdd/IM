package tcp

import (
	"IM/tcp/Message"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestVideoSignalRoutesToPeer(t *testing.T) {
	server := NewServer("", 0, 10*time.Second)
	server.AddHandler(Router)

	senderConn, senderServerConn := net.Pipe()
	defer senderConn.Close()
	defer senderServerConn.Close()
	sender := NewClient(senderServerConn, server)
	sender.setUID("caller")
	go sender.MessageHandler()

	peerConn, peerServerConn := net.Pipe()
	defer peerConn.Close()
	defer peerServerConn.Close()
	peer := NewClient(peerServerConn, server)
	peer.setUID("callee")
	server.Register("callee", peer)

	body, _ := json.Marshal(map[string]any{
		"action":      "video_signal",
		"to_uid":      "callee",
		"signal_type": "offer",
		"sdp":         "v=0\r\n",
		"call_id":     "call-1",
	})
	sender.Process(Message.NewMessage(Message.Json, 1, body))

	f, err := readFrame(peerConn)
	if err != nil {
		t.Fatalf("read video signal frame: %v", err)
	}
	if f.GetMsgType() != Message.Json {
		t.Fatalf("expected Json video signal, got %d", f.GetMsgType())
	}
	var n map[string]any
	if err := json.Unmarshal(f.Data, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n["event"] != "video_signal" || n["from_uid"] != "caller" ||
		n["to_uid"] != "callee" || n["signal_type"] != "offer" ||
		n["sdp"] != "v=0\r\n" || n["call_id"] != "call-1" {
		t.Errorf("unexpected video signal payload: %+v", n)
	}
}
