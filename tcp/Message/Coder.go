package Message

import (
	"encoding/binary"
	"errors"
)

const (
	HeadSize = 8 // 1B type + 3B key + 4B len
)

func Encode(m *Message) []byte {
	total := HeadSize + len(m.Data)
	buf := make([]byte, total)
	buf[0] = m.t
	key := m.key
	buf[1] = byte((key >> 16) & 0xFF)
	buf[2] = byte((key >> 8) & 0xFF)
	buf[3] = byte(key & 0xFF)
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(m.Data)))
	if m.Data != nil {
		copy(buf[8:], m.Data)
	}
	return buf
}

func Decode(data []byte) (*Message, error) {
	if len(data) < HeadSize {
		return nil, errors.New("packet too short")
	}
	var m Message
	m.t = data[0]
	m.key = uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	bodyLen := binary.BigEndian.Uint32(data[4:8])
	if uint32(len(data)) < HeadSize+bodyLen {
		return nil, errors.New("incomplete packet body")
	}
	m.Data = make([]byte, bodyLen)
	copy(m.Data, data[8:8+bodyLen])
	return &m, nil
}

func FullPacketSize(data []byte) (int, bool) {
	// not full header
	if len(data) < HeadSize {
		return 0, false
	}
	bodyLen := binary.BigEndian.Uint32(data[4:8])
	total := HeadSize + int(bodyLen)
	if len(data) < total {
		return total, false
	}
	return total, true
}
