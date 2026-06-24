package utils

import (
	"sync"
	"time"
)

// 雪花算法相关常量定义
const (
	epoch          int64 = 1700000000000 // 2023-11-14 00:00:00 UTC in milliseconds
	workerBits     uint8 = 10             // 机器 ID 占用的位数
	sequenceBits   uint8 = 12             // 同一毫秒内序列号占用的位数
	workerMax      int64 = -1 ^ (-1 << workerBits)   // 机器 ID 最大值
	sequenceMax    int64 = -1 ^ (-1 << sequenceBits) // 序列号最大值
	timeShift             = workerBits + sequenceBits // 时间戳左移位数
	workerShift           = sequenceBits              // 机器 ID 左移位数
)

// Snowflake 雪花算法 ID 生成器，通过互斥锁保证并发安全
type Snowflake struct {
	mu       sync.Mutex
	workerID int64
	sequence int64
	lastMs   int64
}

// defaultNode 默认的雪花 ID 生成节点
var defaultNode *Snowflake

// init 包初始化时创建默认节点，机器 ID 固定为 1
func init() {
	defaultNode = &Snowflake{workerID: 1}
}

// NextID 生成下一个全局唯一的雪花 ID
func (s *Snowflake) NextID() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	// 处理时钟回拨：若当前时间小于上次时间，则沿用上次时间避免 ID 倒退
	if now < s.lastMs {
		now = s.lastMs
	}
	if now == s.lastMs {
		s.sequence = (s.sequence + 1) & sequenceMax
		// 同一毫秒内序列号用尽，自旋等待到下一毫秒
		if s.sequence == 0 {
			for now <= s.lastMs {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}
	s.lastMs = now

	// 按位拼装：时间戳 | 机器 ID | 序列号
	id := ((now - epoch) << timeShift) |
		(s.workerID << workerShift) |
		s.sequence

	return uint64(id)
}

// GenerateId 使用默认节点生成一个全局唯一 ID
func GenerateId() uint64 {
	return defaultNode.NextID()
}
