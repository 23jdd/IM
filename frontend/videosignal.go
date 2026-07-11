package main

const videoSignalAction = "video_signal"

// VideoSignalRequest is the bridge payload for WebRTC signaling frames.
type VideoSignalRequest struct {
	Action     string `json:"action"`
	ToUid      string `json:"to_uid"`
	SignalType string `json:"signal_type"`
	CallID     string `json:"call_id"`
	SDP        string `json:"sdp,omitempty"`
	Candidate  any    `json:"candidate,omitempty"`
}
