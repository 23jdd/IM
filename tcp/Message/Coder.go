package Message

import (
	"encoding/binary"
	"errors"
)

// 帧头长度常量。
const (
	HeadSize = 8 // 1B type + 3B key + 4B len
)

// Encode 将消息编码为字节流：1B 类型 + 3B key + 4B 长度 + 变长消息体。
func Encode(m *Message) []byte {
	total := HeadSize + len(m.Data)
	buf := make([]byte, total)
	buf[0] = m.t
	key := m.key
	// key 为 24bit，按大端拆成 3 字节写入。
	buf[1] = byte((key >> 16) & 0xFF)
	buf[2] = byte((key >> 8) & 0xFF)
	buf[3] = byte(key & 0xFF)
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(m.Data)))
	if m.Data != nil {
		copy(buf[8:], m.Data)
	}
	return buf
}

// Decode 从字节流解码出消息；数据不足或包体不完整时返回错误。
func Decode(data []byte) (*Message, error) {
	if len(data) < HeadSize {
		return nil, errors.New("packet too short")
	}
	var m Message
	m.t = data[0]
	// 还原 24bit key。
	m.key = uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	bodyLen := binary.BigEndian.Uint32(data[4:8])
	// 实际数据不足声明的包体长度，视为不完整包。
	if uint32(len(data)) < HeadSize+bodyLen {
		return nil, errors.New("incomplete packet body")
	}
	if bodyLen == 0 {
		return &m, nil
	}
	m.Data = make([]byte, bodyLen)
	copy(m.Data, data[8:8+bodyLen])
	m.len = bodyLen
	return &m, nil
}

// FullPacketSize 探测缓冲区中是否含一个完整包，返回所需总长度及是否已完整。
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
