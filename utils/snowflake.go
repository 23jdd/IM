package utils

import (
	"sync"
	"time"
)

const (
	epoch          int64 = 1700000000000 // 2023-11-14 00:00:00 UTC in milliseconds
	workerBits     uint8 = 10
	sequenceBits   uint8 = 12
	workerMax      int64 = -1 ^ (-1 << workerBits)
	sequenceMax    int64 = -1 ^ (-1 << sequenceBits)
	timeShift             = workerBits + sequenceBits
	workerShift           = sequenceBits
)

type Snowflake struct {
	mu       sync.Mutex
	workerID int64
	sequence int64
	lastMs   int64
}

var defaultNode *Snowflake

func init() {
	defaultNode = &Snowflake{workerID: 1}
}

func (s *Snowflake) NextID() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	if now < s.lastMs {
		now = s.lastMs
	}
	if now == s.lastMs {
		s.sequence = (s.sequence + 1) & sequenceMax
		if s.sequence == 0 {
			for now <= s.lastMs {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}
	s.lastMs = now

	id := ((now - epoch) << timeShift) |
		(s.workerID << workerShift) |
		s.sequence

	return uint64(id)
}

func GenerateId() uint64 {
	return defaultNode.NextID()
}
