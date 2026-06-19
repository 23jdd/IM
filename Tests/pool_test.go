package Tests

import (
	"IM/tcp"
	"testing"
)

func TestNewTieredPool(t *testing.T) {
	tp := tcp.NewTieredPool(8, 64, 256)
	if tp == nil {
		t.Fatal("NewTieredPool returned nil")
	}
}

func TestNewTieredPoolPanicsEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty capacities")
		}
	}()
	tcp.NewTieredPool()
}

func TestTieredPoolGet(t *testing.T) {
	tp := tcp.NewTieredPool(8, 64, 256, 1024)

	tests := []struct {
		name     string
		size     int
		wantLen  int
	}{
		{"small", 4, 4},
		{"exact_match_8", 8, 8},
		{"medium", 50, 50},
		{"exact_match_64", 64, 64},
		{"large", 200, 200},
		{"exact_match_256", 256, 256},
		{"exact_match_1024", 1024, 1024},
		{"larger_than_all", 2048, 2048},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := tp.Get(tt.size)
			if len(buf) != tt.wantLen {
				t.Errorf("Get(%d) length = %d, want %d", tt.size, len(buf), tt.wantLen)
			}
		})
	}
}

func TestTieredPoolGetCapacity(t *testing.T) {
	tp := tcp.NewTieredPool(64, 256, 1024)

	buf := tp.Get(32)
	if cap(buf) < 64 {
		t.Errorf("Get(32) should have capacity at least 64, got %d", cap(buf))
	}

	buf = tp.Get(200)
	if cap(buf) < 256 {
		t.Errorf("Get(200) should have capacity at least 256, got %d", cap(buf))
	}
}

func TestTieredPoolPutAndReuse(t *testing.T) {
	tp := tcp.NewTieredPool(64, 256)

	buf1 := tp.Get(32)
	ptr1 := &buf1[0]
	tp.Put(buf1)

	buf2 := tp.Get(32)
	ptr2 := &buf2[0]

	if ptr1 != ptr2 {
		t.Error("expected buffer reuse, but got different pointers")
	}
}

func TestTieredPoolPutReturnsToCorrectPool(t *testing.T) {
	tp := tcp.NewTieredPool(8, 64, 256)

	buf := tp.Get(50)
	capBefore := cap(buf)
	tp.Put(buf)

	buf2 := tp.Get(50)
	if cap(buf2) < capBefore {
		t.Errorf("expected capacity >= %d, got %d", capBefore, cap(buf2))
	}
}

func TestTieredPoolPutDiscardsOversized(t *testing.T) {
	tp := tcp.NewTieredPool(8, 64)

	buf := make([]byte, 1024)
	tp.Put(buf)
	// This shouldn't panic; oversized buffers are simply discarded.
}

func TestTieredPoolGetZero(t *testing.T) {
	tp := tcp.NewTieredPool(8, 64)
	buf := tp.Get(0)
	if len(buf) != 0 {
		t.Errorf("Get(0) length = %d, want 0", len(buf))
	}
}

func TestTieredPoolMultipleGets(t *testing.T) {
	tp := tcp.NewTieredPool(64, 256)

	for i := 0; i < 100; i++ {
		buf := tp.Get(48)
		if len(buf) != 48 {
			t.Errorf("iteration %d: length = %d, want 48", i, len(buf))
		}
		tp.Put(buf)
	}
}
