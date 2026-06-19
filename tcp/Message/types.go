package Message

type MsgType = byte

const (
	ACK byte = iota
	Nack
	Auth
	HeartBeat
	Json
	Text
	Blob
)
