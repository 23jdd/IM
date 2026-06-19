package Message

type MsgType = byte

const (
	ACK byte = iota
	HeartBeat
	Json
	Text
	Blob
)
