package namestore

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestClient_Incr tests atomic increment operations.
func TestClient_Incr(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Incr on new key.
	val, err := c.Incr(ctx, "counter", 1)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}

	if val != 1 {
		t.Errorf("Incr result = %d, want 1", val)
	}

	// Incr again.
	val, err = c.Incr(ctx, "counter", 5)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}

	if val != 6 {
		t.Errorf("Incr result = %d, want 6", val)
	}
}

// TestClient_Incr_TypeMismatch tests Incr on non-integer values.
func TestClient_Incr_TypeMismatch(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set non-integer value.
	_ = c.Set(ctx, "key1", []byte("not-a-number"), 0)

	_, err := c.Incr(ctx, "key1", 1)
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("Incr on non-integer: expected ErrTypeMismatch, got %v", err)
	}
}

// TestClient_Decr tests atomic decrement operations.
func TestClient_Decr(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Initialize counter.
	_, _ = c.Incr(ctx, "counter", 10)

	// Decr.
	val, err := c.Decr(ctx, "counter", 3)
	if err != nil {
		t.Fatalf("Decr failed: %v", err)
	}

	if val != 7 {
		t.Errorf("Decr result = %d, want 7", val)
	}
}

// TestClient_GetSet tests atomic get-and-set operations.
func TestClient_GetSet(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// GetSet on new key.
	old, err := c.GetSet(ctx, "key1", []byte("new-value"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetSet on new key: expected ErrNotFound, got %v", err)
	}

	// Verify new value was set.
	data, _ := c.Get(ctx, "key1")
	if string(data) != "new-value" {
		t.Errorf("GetSet set value = %q, want %q", data, "new-value")
	}

	// GetSet on existing key.
	old, err = c.GetSet(ctx, "key1", []byte("newer-value"))
	if err != nil {
		t.Fatalf("GetSet on existing key failed: %v", err)
	}

	if string(old) != "new-value" {
		t.Errorf("GetSet returned old value = %q, want %q", old, "new-value")
	}

	// Verify new value.
	data, _ = c.Get(ctx, "key1")
	if string(data) != "newer-value" {
		t.Errorf("GetSet set value = %q, want %q", data, "newer-value")
	}
}

// TestClient_CompareAndSwap tests atomic compare-and-swap operations.
func TestClient_CompareAndSwap(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set initial value.
	_ = c.Set(ctx, "key1", []byte("old-value"), 0)

	// CAS with correct old value.
	ok, err := c.CompareAndSwap(ctx, "key1", []byte("old-value"), []byte("new-value"), 0)
	if err != nil {
		t.Fatalf("CompareAndSwap failed: %v", err)
	}

	if !ok {
		t.Error("CompareAndSwap should succeed with correct old value")
	}

	// Verify new value.
	data, _ := c.Get(ctx, "key1")
	if string(data) != "new-value" {
		t.Errorf("CompareAndSwap set value = %q, want %q", data, "new-value")
	}

	// CAS with incorrect old value.
	ok, err = c.CompareAndSwap(ctx, "key1", []byte("wrong-value"), []byte("another-value"), 0)
	if err != nil {
		t.Fatalf("CompareAndSwap failed: %v", err)
	}

	if ok {
		t.Error("CompareAndSwap should fail with incorrect old value")
	}

	// Value should remain unchanged.
	data, _ = c.Get(ctx, "key1")
	if string(data) != "new-value" {
		t.Errorf("CompareAndSwap should not modify value = %q, want %q", data, "new-value")
	}
}

// TestClient_CompareAndSwap_MissingKey tests CAS on non-existent keys.
func TestClient_CompareAndSwap_MissingKey(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	ok, err := c.CompareAndSwap(ctx, "missing", []byte("old"), []byte("new"), 0)
	if err != nil {
		t.Fatalf("CompareAndSwap on missing key failed: %v", err)
	}

	if ok {
		t.Error("CompareAndSwap should fail on missing key")
	}
}

// TestClient_CompareAndSwap_WithTTL tests CAS with TTL.
func TestClient_CompareAndSwap_WithTTL(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set initial value.
	_ = c.Set(ctx, "key1", []byte("old-value"), 0)

	// CAS with TTL.
	ok, err := c.CompareAndSwap(ctx, "key1", []byte("old-value"), []byte("new-value"), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("CompareAndSwap failed: %v", err)
	}

	if !ok {
		t.Error("CompareAndSwap should succeed")
	}

	// Wait for expiration.
	time.Sleep(20 * time.Millisecond)

	exists, _ := c.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should expire after CAS with TTL")
	}
}
