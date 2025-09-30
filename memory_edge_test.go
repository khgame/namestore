package namestore

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

// TestMemoryDriver_Incr_Overflow tests integer overflow behavior.
func TestMemoryDriver_Incr_Overflow(t *testing.T) {
	d := NewInMemoryDriver()
	ctx := context.Background()

	// Test positive overflow.
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(9223372036854775807)) // max int64
	_ = d.Set(ctx, "key1", buf, 0)

	val, err := d.Incr(ctx, "key1", 1)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}

	// Should wrap around due to uint64 conversion.
	expected := int64(-9223372036854775808)
	if val != expected {
		t.Errorf("Incr overflow = %d, want %d", val, expected)
	}
}

// TestMemoryDriver_Keys_InvalidPattern tests pattern validation.
func TestMemoryDriver_Keys_InvalidPattern(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	_ = d.Set(ctx, "prefix:key1", []byte("value1"), 0)

	_, err := d.Keys(ctx, "prefix", "[invalid")
	if !errors.Is(err, ErrInvalidPattern) {
		t.Errorf("Keys with invalid pattern: expected ErrInvalidPattern, got %v", err)
	}
}

// TestMemoryDriver_TTL_ExpiredKey tests TTL on expired keys.
func TestMemoryDriver_TTL_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	_ = d.Set(ctx, "key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, err := d.TTL(ctx, "key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("TTL on expired key: expected ErrNotFound, got %v", err)
	}

	// Verify expired key was deleted.
	if _, ok := d.data["key1"]; ok {
		t.Error("TTL should delete expired key")
	}
}

// TestMemoryDriver_Expire_ExpiredKey tests Expire on expired keys.
func TestMemoryDriver_Expire_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	_ = d.Set(ctx, "key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	err := d.Expire(ctx, "key1", 10*time.Second)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expire on expired key: expected ErrNotFound, got %v", err)
	}

	// Verify expired key was deleted.
	if _, ok := d.data["key1"]; ok {
		t.Error("Expire should delete expired key")
	}
}

// TestMemoryDriver_Persist_ExpiredKey tests Persist on expired keys.
func TestMemoryDriver_Persist_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	_ = d.Set(ctx, "key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	err := d.Persist(ctx, "key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Persist on expired key: expected ErrNotFound, got %v", err)
	}

	// Verify expired key was deleted.
	if _, ok := d.data["key1"]; ok {
		t.Error("Persist should delete expired key")
	}
}

// TestMemoryDriver_Keys_NoPrefix tests Keys with prefix filtering.
func TestMemoryDriver_Keys_NoPrefix(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	_ = d.Set(ctx, "prefix:key1", []byte("value1"), 0)
	_ = d.Set(ctx, "other:key2", []byte("value2"), 0)

	keys, err := d.Keys(ctx, "prefix", "*")
	if err != nil {
		t.Fatalf("Keys failed: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Keys returned %d results, want 1", len(keys))
	}

	if keys[0] != "prefix:key1" {
		t.Errorf("Keys returned %q, want %q", keys[0], "prefix:key1")
	}
}

// TestMemoryDriver_Incr_ExpiredKey tests Incr on expired keys.
func TestMemoryDriver_Incr_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	// Set expired key.
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, 100)
	_ = d.Set(ctx, "key1", buf, 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	// Incr on expired key should start from 0.
	val, err := d.Incr(ctx, "key1", 5)
	if err != nil {
		t.Fatalf("Incr on expired key failed: %v", err)
	}

	if val != 5 {
		t.Errorf("Incr on expired key = %d, want 5", val)
	}
}

// TestMemoryDriver_GetSet_ExpiredKey tests GetSet on expired keys.
func TestMemoryDriver_GetSet_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	_ = d.Set(ctx, "key1", []byte("old"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, err := d.GetSet(ctx, "key1", []byte("new"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetSet on expired key: expected ErrNotFound, got %v", err)
	}

	// Should have set new value.
	data, _ := d.Get(ctx, "key1")
	if string(data) != "new" {
		t.Errorf("GetSet should set new value = %q, want %q", data, "new")
	}
}

// TestMemoryDriver_CompareAndSwap_ExpiredKey tests CAS on expired keys.
func TestMemoryDriver_CompareAndSwap_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	_ = d.Set(ctx, "key1", []byte("old"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	ok, err := d.CompareAndSwap(ctx, "key1", []byte("old"), []byte("new"), 0)
	if err != nil {
		t.Fatalf("CompareAndSwap on expired key failed: %v", err)
	}

	if ok {
		t.Error("CompareAndSwap on expired key should return false")
	}

	// Verify expired key was deleted.
	if _, ok := d.data["key1"]; ok {
		t.Error("CompareAndSwap should delete expired key")
	}
}
