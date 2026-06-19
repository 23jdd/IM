package Message

type Message struct {
	t    MsgType
	key  uint64
	len  int
	data []byte
}

func NewMessage(t MsgType, key uint64, data []byte) *Message {
	return &Message{
		t:    t,
		key:  key,
		len:  len(data),
		data: nil,
	}
}
func AckMessage(key uint64) *Message {
	return NewMessage(ACK, key, nil)
}
func HeartMessage(key uint64) *Message {
	return NewMessage(HeartBeat, key, nil)
}
