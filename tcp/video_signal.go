package tcp

import (
	"IM/tcp/Message"
	"encoding/json"
)

const realtimeActionVideoSignal = "video_signal"

// RealtimeSignalRequest is the JSON control frame used for lightweight realtime events.
type RealtimeSignalRequest struct {
	Action     string          `json:"action"`
	ToUid      string          `json:"to_uid"`
	GroupId    string          `json:"group_id"`
	UpTo       int64           `json:"up_to"`
	SignalType string          `json:"signal_type"`
	SDP        string          `json:"sdp"`
	Candidate  json.RawMessage `json:"candidate"`
	CallID     string          `json:"call_id"`
}

// VideoSignalPayload is forwarded to the callee through the existing Json channel.
type VideoSignalPayload struct {
	Event      string `json:"event"`
	FromUid    string `json:"from_uid"`
	ToUid      string `json:"to_uid"`
	SignalType string `json:"signal_type"`
	CallID     string `json:"call_id"`
	SDP        string `json:"sdp,omitempty"`
	Candidate  any    `json:"candidate,omitempty"`
}

func buildVideoSignalPayload(fromUid string, req RealtimeSignalRequest) (VideoSignalPayload, bool) {
	if req.ToUid == "" || req.SignalType == "" {
		return VideoSignalPayload{}, false
	}
	payload := VideoSignalPayload{
		Event:      realtimeActionVideoSignal,
		FromUid:    fromUid,
		ToUid:      req.ToUid,
		SignalType: req.SignalType,
		CallID:     req.CallID,
		SDP:        req.SDP,
	}
	if len(req.Candidate) > 0 && string(req.Candidate) != "null" {
		var v any
		if err := json.Unmarshal(req.Candidate, &v); err == nil {
			payload.Candidate = v
		}
	}
	return payload, true
}

// handleVideoSignal forwards WebRTC signaling between single-chat peers.
// Media flows peer-to-peer; the IM server only relays offer/answer/ICE metadata.
func handleVideoSignal(c *Client, req RealtimeSignalRequest) {
	payload, ok := buildVideoSignalPayload(c.UID(), req)
	if !ok {
		return
	}
	data, _ := json.Marshal(payload)
	_ = c.server.RouteTo(req.ToUid, Message.NewMessage(Message.Json, 0, data))
}
