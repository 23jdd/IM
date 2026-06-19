package tcp

import "sync"

// TieredPool is a collection of sync.Pools of different capacities,
// designed to reuse []byte slices of varying sizes while minimizing memory waste.
type TieredPool struct {
	caps  []int
	pools []sync.Pool
}

// New creates a new TieredPool with the given capacities.
// Each capacity defines a pool of buffers with that exact capacity.
// The capacities slice must be sorted in ascending order.
func New(capacities []int) *TieredPool {
	if len(capacities) == 0 {
		panic("tiered buffer: capacities must not be empty")
	}
	tp := &TieredPool{
		caps:  capacities,
		pools: make([]sync.Pool, len(capacities)),
	}
	for i, c := range capacities {
		c := c
		tp.pools[i].New = func() any {
			return make([]byte, 0, c)
		}
	}
	return tp
}

// Get returns a []byte of length size with capacity at least size.
// The buffer is taken from the smallest pool whose capacity >= size.
// If no pool is large enough, a new buffer is allocated without pooling.
func (tp *TieredPool) Get(size int) []byte {
	for i, c := range tp.caps {
		if c >= size {
			buf := tp.pools[i].Get().([]byte)
			if cap(buf) < size {
				// Should never happen under normal use, but be defensive.
				tp.pools[i].Put(buf[:0])
				return make([]byte, size)
			}
			return buf[:size]
		}
	}
	return make([]byte, size)
}

// Put returns a buffer to the pool. The buffer's capacity determines which
// pool it goes into: it's placed into the smallest pool whose capacity >= cap(buf).
// If the capacity exceeds the largest pool's capacity, the buffer is discarded.
func (tp *TieredPool) Put(buf []byte) {
	c := cap(buf)
	for i, capa := range tp.caps {
		if c <= capa {
			tp.pools[i].Put(buf[:0])
			return
		}
	}
	// Discard: capacity too large.
}
