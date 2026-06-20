package main

import (
	"encoding/binary"
	"errors"
)

// 二进制协议（与后端 tcp/Message 一致，自包含实现，不依赖后端 module）
//
// ┌──────┬──────────┬──────────┬──────────┐
// │ 1B   │ 3B       │ 4B       │ N bytes  │
// │ type │ key      │ body len │ body     │
// └──────┴──────────┴──────────┴──────────┘

const headSize = 8

const (
	msgACK       byte = 0 // 确认
	msgNack      byte = 1 // 拒绝
	msgAuth      byte = 2 // JWT 认证
	msgHeartBeat byte = 3 // 心跳
	msgJson      byte = 4 // 系统消息（触发离线同步）
	msgText      byte = 5 // 文本聊天
	msgBlob      byte = 6 // 二进制（离线消息 JSON）
)

type frame struct {
	Type byte
	Key  uint32
	Data []byte
}

func encodeFrame(t byte, key uint32, data []byte) []byte {
	buf := make([]byte, headSize+len(data))
	buf[0] = t
	buf[1] = byte((key >> 16) & 0xFF)
	buf[2] = byte((key >> 8) & 0xFF)
	buf[3] = byte(key & 0xFF)
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(data)))
	if len(data) > 0 {
		copy(buf[8:], data)
	}
	return buf
}

func parseHeader(head []byte) (t byte, key uint32, bodyLen uint32, err error) {
	if len(head) < headSize {
		return 0, 0, 0, errors.New("header too short")
	}
	t = head[0]
	key = uint32(head[1])<<16 | uint32(head[2])<<8 | uint32(head[3])
	bodyLen = binary.BigEndian.Uint32(head[4:8])
	return t, key, bodyLen, nil
}
