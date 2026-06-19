package Tests

import (
	"IM/tcp"
	"testing"
)

func TestNewContext(t *testing.T) {
	ctx := tcp.NewContext()
	if ctx == nil {
		t.Fatal("NewContext returned nil")
	}
}

func TestContextSetAndGet(t *testing.T) {
	ctx := tcp.NewContext()

	ctx.Set("uid", uint32(12345))
	val, ok := ctx.Get("uid")
	if !ok {
		t.Fatal("expected key 'uid' to exist")
	}
	if val.(uint32) != 12345 {
		t.Errorf("got %v, want 12345", val)
	}
}

func TestContextGetMissingKey(t *testing.T) {
	ctx := tcp.NewContext()

	val, ok := ctx.Get("nonexistent")
	if ok {
		t.Error("expected key 'nonexistent' to not exist")
	}
	if val != nil {
		t.Errorf("expected nil value, got %v", val)
	}
}

func TestContextOverwrite(t *testing.T) {
	ctx := tcp.NewContext()

	ctx.Set("key", "first")
	ctx.Set("key", "second")

	val, ok := ctx.Get("key")
	if !ok {
		t.Fatal("expected key 'key' to exist")
	}
	if val.(string) != "second" {
		t.Errorf("got %v, want 'second'", val)
	}
}

func TestContextStringValue(t *testing.T) {
	ctx := tcp.NewContext()

	ctx.Set("name", "test_user")
	val, ok := ctx.Get("name")
	if !ok {
		t.Fatal("expected key 'name' to exist")
	}
	if val.(string) != "test_user" {
		t.Errorf("got %v, want 'test_user'", val)
	}
}

func TestContextBoolValue(t *testing.T) {
	ctx := tcp.NewContext()

	ctx.Set("online", true)
	val, ok := ctx.Get("online")
	if !ok {
		t.Fatal("expected key 'online' to exist")
	}
	if val.(bool) != true {
		t.Errorf("got %v, want true", val)
	}
}

func TestContextMultipleKeys(t *testing.T) {
	ctx := tcp.NewContext()

	ctx.Set("a", 1)
	ctx.Set("b", "hello")
	ctx.Set("c", 3.14)

	if v, ok := ctx.Get("a"); !ok || v.(int) != 1 {
		t.Errorf("key 'a': got %v, want 1", v)
	}
	if v, ok := ctx.Get("b"); !ok || v.(string) != "hello" {
		t.Errorf("key 'b': got %v, want 'hello'", v)
	}
	if v, ok := ctx.Get("c"); !ok || v.(float64) != 3.14 {
		t.Errorf("key 'c': got %v, want 3.14", v)
	}
}

func TestContextDel(t *testing.T) {
	ctx := tcp.NewContext()

	ctx.Set("key", "value")
	_, ok := ctx.Get("key")
	if !ok {
		t.Fatal("expected key to exist before delete")
	}

	ctx.Del("key")
	_, ok = ctx.Get("key")
	if ok {
		t.Error("expected key to not exist after delete")
	}
}

func TestContextDelNonExistent(t *testing.T) {
	ctx := tcp.NewContext()
	ctx.Del("nonexistent")
	// should not panic
}
