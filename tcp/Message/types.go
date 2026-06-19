package Message

type MsgType = byte

const (
	ACK byte = iota
	Json
	Text
	Blob
)
