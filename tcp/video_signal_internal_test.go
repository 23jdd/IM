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
		"action":      realtimeActionVideoSignal,
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
	if n["event"] != realtimeActionVideoSignal || n["from_uid"] != "caller" ||
		n["to_uid"] != "callee" || n["signal_type"] != "offer" ||
		n["sdp"] != "v=0\r\n" || n["call_id"] != "call-1" {
		t.Errorf("unexpected video signal payload: %+v", n)
	}
}

func TestBuildVideoSignalPayloadWithCandidate(t *testing.T) {
	payload, ok := buildVideoSignalPayload("caller", RealtimeSignalRequest{
		ToUid:      "callee",
		SignalType: "candidate",
		CallID:     "call-2",
		Candidate:  json.RawMessage(`{"candidate":"candidate:1","sdpMid":"0","sdpMLineIndex":0}`),
	})
	if !ok {
		t.Fatal("expected candidate signal to be accepted")
	}
	if payload.Event != realtimeActionVideoSignal || payload.FromUid != "caller" ||
		payload.ToUid != "callee" || payload.SignalType != "candidate" || payload.CallID != "call-2" {
		t.Fatalf("unexpected payload metadata: %+v", payload)
	}
	candidate, ok := payload.Candidate.(map[string]any)
	if !ok {
		t.Fatalf("expected candidate map, got %T", payload.Candidate)
	}
	if candidate["candidate"] != "candidate:1" || candidate["sdpMid"] != "0" || candidate["sdpMLineIndex"] != float64(0) {
		t.Fatalf("unexpected candidate payload: %+v", candidate)
	}
}

func TestBuildVideoSignalPayloadRejectsIncompleteRequest(t *testing.T) {
	cases := []RealtimeSignalRequest{
		{SignalType: "offer"},
		{ToUid: "callee"},
	}
	for _, tc := range cases {
		if payload, ok := buildVideoSignalPayload("caller", tc); ok {
			t.Fatalf("expected incomplete request to be rejected, got %+v", payload)
		}
	}
}
