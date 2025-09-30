package namestore

import (
	"context"
	"encoding/binary"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"
)

// ========== Batch Operations Tests ==========

func TestClient_MGet(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data
	c.Set(ctx, "key1", []byte("value1"), 0)
	c.Set(ctx, "key2", []byte("value2"), 0)

	result, err := c.MGet(ctx, "key1", "key2", "missing")
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("MGet returned %d results, want 2", len(result))
	}

	if string(result["key1"]) != "value1" {
		t.Errorf("MGet key1 = %q, want %q", result["key1"], "value1")
	}

	if string(result["key2"]) != "value2" {
		t.Errorf("MGet key2 = %q, want %q", result["key2"], "value2")
	}

	if _, ok := result["missing"]; ok {
		t.Error("MGet should not return missing key")
	}
}

func TestClient_MGet_Empty(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	result, err := c.MGet(ctx)
	if err != nil {
		t.Fatalf("MGet with empty keys failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("MGet with empty keys returned %d results, want 0", len(result))
	}
}

func TestClient_MSet(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	pairs := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	err := c.MSet(ctx, pairs, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Verify all keys were set
	for k, expected := range pairs {
		data, err := c.Get(ctx, k)
		if err != nil {
			t.Errorf("Get %s failed: %v", k, err)
		}
		if string(data) != string(expected) {
			t.Errorf("Get %s = %q, want %q", k, data, expected)
		}
	}
}

func TestClient_MSet_WithTTL(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	pairs := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	err := c.MSet(ctx, pairs, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Verify keys exist
	exists, _ := c.Exists(ctx, "key1")
	if !exists {
		t.Error("key1 should exist after MSet")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	exists, _ = c.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should expire after TTL")
	}
}

func TestClient_MSet_Empty(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	err := c.MSet(ctx, map[string][]byte{}, 0)
	if err != nil {
		t.Fatalf("MSet with empty pairs failed: %v", err)
	}
}

func TestClient_MDel(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data
	c.Set(ctx, "key1", []byte("value1"), 0)
	c.Set(ctx, "key2", []byte("value2"), 0)
	c.Set(ctx, "key3", []byte("value3"), 0)

	err := c.MDel(ctx, "key1", "key2")
	if err != nil {
		t.Fatalf("MDel failed: %v", err)
	}

	// Verify deletion
	exists, _ := c.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should be deleted")
	}

	exists, _ = c.Exists(ctx, "key2")
	if exists {
		t.Error("key2 should be deleted")
	}

	exists, _ = c.Exists(ctx, "key3")
	if !exists {
		t.Error("key3 should still exist")
	}
}

func TestClient_MDel_Empty(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	err := c.MDel(ctx)
	if err != nil {
		t.Fatalf("MDel with empty keys failed: %v", err)
	}
}

// ========== TTL Management Tests ==========

func TestClient_TTL(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Key with TTL
	c.Set(ctx, "key1", []byte("value1"), 100*time.Millisecond)
	ttl, err := c.TTL(ctx, "key1")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl <= 0 || ttl > 100*time.Millisecond {
		t.Errorf("TTL = %v, expected > 0 and <= 100ms", ttl)
	}

	// Key without TTL
	c.Set(ctx, "key2", []byte("value2"), 0)
	ttl, err = c.TTL(ctx, "key2")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl != -1 {
		t.Errorf("TTL for key without expiration = %v, want -1", ttl)
	}

	// Missing key
	_, err = c.TTL(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("TTL for missing key: expected ErrNotFound, got %v", err)
	}
}

func TestClient_Expire(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set key without TTL
	c.Set(ctx, "key1", []byte("value1"), 0)

	// Add expiration
	err := c.Expire(ctx, "key1", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Expire failed: %v", err)
	}

	// Verify key exists
	exists, _ := c.Exists(ctx, "key1")
	if !exists {
		t.Error("key1 should exist after Expire")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	exists, _ = c.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should expire after TTL")
	}
}

func TestClient_Expire_MissingKey(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	err := c.Expire(ctx, "missing", 10*time.Second)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expire for missing key: expected ErrNotFound, got %v", err)
	}
}

func TestClient_Persist(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set key with TTL
	c.Set(ctx, "key1", []byte("value1"), 10*time.Millisecond)

	// Remove expiration
	err := c.Persist(ctx, "key1")
	if err != nil {
		t.Fatalf("Persist failed: %v", err)
	}

	// Wait longer than original TTL
	time.Sleep(20 * time.Millisecond)

	// Key should still exist
	exists, _ := c.Exists(ctx, "key1")
	if !exists {
		t.Error("key1 should still exist after Persist")
	}

	// Verify TTL is -1
	ttl, _ := c.TTL(ctx, "key1")
	if ttl != -1 {
		t.Errorf("TTL after Persist = %v, want -1", ttl)
	}
}

func TestClient_Persist_MissingKey(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	err := c.Persist(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Persist for missing key: expected ErrNotFound, got %v", err)
	}
}

// ========== Namespace Operations Tests ==========

func TestClient_Keys(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data
	c.Set(ctx, "user:1", []byte("alice"), 0)
	c.Set(ctx, "user:2", []byte("bob"), 0)
	c.Set(ctx, "admin:1", []byte("charlie"), 0)

	// Get all keys
	keys, err := c.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("Keys failed: %v", err)
	}

	sort.Strings(keys)
	expected := []string{"admin:1", "user:1", "user:2"}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Keys = %v, want %v", keys, expected)
	}
}

func TestClient_Keys_WithPattern(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data
	c.Set(ctx, "user:1", []byte("alice"), 0)
	c.Set(ctx, "user:2", []byte("bob"), 0)
	c.Set(ctx, "admin:1", []byte("charlie"), 0)

	// Get keys matching pattern
	keys, err := c.Keys(ctx, "user:*")
	if err != nil {
		t.Fatalf("Keys with pattern failed: %v", err)
	}

	sort.Strings(keys)
	expected := []string{"user:1", "user:2"}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Keys with pattern = %v, want %v", keys, expected)
	}
}

func TestClient_Keys_Empty(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	keys, err := c.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("Keys on empty namespace failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Keys on empty namespace = %v, want empty", keys)
	}
}

func TestClient_Clear(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data
	c.Set(ctx, "key1", []byte("value1"), 0)
	c.Set(ctx, "key2", []byte("value2"), 0)

	err := c.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify all keys are deleted
	exists, _ := c.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should be cleared")
	}

	exists, _ = c.Exists(ctx, "key2")
	if exists {
		t.Error("key2 should be cleared")
	}
}

func TestClient_Clear_IsolatesNamespaces(t *testing.T) {
	c1 := New[string]("root", "domain1")
	c2 := New[string]("root", "domain2")
	ctx := context.Background()

	// Set up test data in both namespaces
	c1.Set(ctx, "key1", []byte("value1"), 0)
	c2.Set(ctx, "key1", []byte("value2"), 0)

	// Clear only domain1
	c1.Clear(ctx)

	// domain1 should be empty
	exists, _ := c1.Exists(ctx, "key1")
	if exists {
		t.Error("domain1 key1 should be cleared")
	}

	// domain2 should be untouched
	exists, _ = c2.Exists(ctx, "key1")
	if !exists {
		t.Error("domain2 key1 should still exist")
	}
}

// ========== Atomic Operations Tests ==========

func TestClient_Incr(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Incr on new key
	val, err := c.Incr(ctx, "counter", 1)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}

	if val != 1 {
		t.Errorf("Incr result = %d, want 1", val)
	}

	// Incr again
	val, err = c.Incr(ctx, "counter", 5)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}

	if val != 6 {
		t.Errorf("Incr result = %d, want 6", val)
	}
}

func TestClient_Incr_TypeMismatch(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set non-integer value
	c.Set(ctx, "key1", []byte("not-a-number"), 0)

	_, err := c.Incr(ctx, "key1", 1)
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("Incr on non-integer: expected ErrTypeMismatch, got %v", err)
	}
}

func TestClient_Decr(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Initialize counter
	c.Incr(ctx, "counter", 10)

	// Decr
	val, err := c.Decr(ctx, "counter", 3)
	if err != nil {
		t.Fatalf("Decr failed: %v", err)
	}

	if val != 7 {
		t.Errorf("Decr result = %d, want 7", val)
	}
}

func TestClient_GetSet(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// GetSet on new key
	old, err := c.GetSet(ctx, "key1", []byte("new-value"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetSet on new key: expected ErrNotFound, got %v", err)
	}

	// Verify new value was set
	data, _ := c.Get(ctx, "key1")
	if string(data) != "new-value" {
		t.Errorf("GetSet set value = %q, want %q", data, "new-value")
	}

	// GetSet on existing key
	old, err = c.GetSet(ctx, "key1", []byte("newer-value"))
	if err != nil {
		t.Fatalf("GetSet on existing key failed: %v", err)
	}

	if string(old) != "new-value" {
		t.Errorf("GetSet returned old value = %q, want %q", old, "new-value")
	}

	// Verify new value
	data, _ = c.Get(ctx, "key1")
	if string(data) != "newer-value" {
		t.Errorf("GetSet set value = %q, want %q", data, "newer-value")
	}
}

func TestClient_CompareAndSwap(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set initial value
	c.Set(ctx, "key1", []byte("old-value"), 0)

	// CAS with correct old value
	ok, err := c.CompareAndSwap(ctx, "key1", []byte("old-value"), []byte("new-value"), 0)
	if err != nil {
		t.Fatalf("CompareAndSwap failed: %v", err)
	}

	if !ok {
		t.Error("CompareAndSwap should succeed with correct old value")
	}

	// Verify new value
	data, _ := c.Get(ctx, "key1")
	if string(data) != "new-value" {
		t.Errorf("CompareAndSwap set value = %q, want %q", data, "new-value")
	}

	// CAS with incorrect old value
	ok, err = c.CompareAndSwap(ctx, "key1", []byte("wrong-value"), []byte("another-value"), 0)
	if err != nil {
		t.Fatalf("CompareAndSwap failed: %v", err)
	}

	if ok {
		t.Error("CompareAndSwap should fail with incorrect old value")
	}

	// Value should remain unchanged
	data, _ = c.Get(ctx, "key1")
	if string(data) != "new-value" {
		t.Errorf("CompareAndSwap should not modify value = %q, want %q", data, "new-value")
	}
}

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

func TestClient_CompareAndSwap_WithTTL(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set initial value
	c.Set(ctx, "key1", []byte("old-value"), 0)

	// CAS with TTL
	ok, err := c.CompareAndSwap(ctx, "key1", []byte("old-value"), []byte("new-value"), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("CompareAndSwap failed: %v", err)
	}

	if !ok {
		t.Error("CompareAndSwap should succeed")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	exists, _ := c.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should expire after CAS with TTL")
	}
}

// ========== Memory Driver Extended Tests ==========

func TestMemoryDriver_Incr_Overflow(t *testing.T) {
	d := NewInMemoryDriver()
	ctx := context.Background()

	// Test positive overflow
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(9223372036854775807)) // max int64
	d.Set(ctx, "key1", buf, 0)

	val, err := d.Incr(ctx, "key1", 1)
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}

	// Should wrap around due to uint64 conversion
	expected := int64(-9223372036854775808)
	if val != expected {
		t.Errorf("Incr overflow = %d, want %d", val, expected)
	}
}

func TestMemoryDriver_Keys_InvalidPattern(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "prefix:key1", []byte("value1"), 0)

	_, err := d.Keys(ctx, "prefix", "[invalid")
	if !errors.Is(err, ErrInvalidPattern) {
		t.Errorf("Keys with invalid pattern: expected ErrInvalidPattern, got %v", err)
	}
}

// ========== Additional Edge Cases for 100% Coverage ==========

func TestMemoryDriver_TTL_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, err := d.TTL(ctx, "key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("TTL on expired key: expected ErrNotFound, got %v", err)
	}

	// Verify expired key was deleted
	if _, ok := d.data["key1"]; ok {
		t.Error("TTL should delete expired key")
	}
}

func TestMemoryDriver_Expire_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	err := d.Expire(ctx, "key1", 10*time.Second)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expire on expired key: expected ErrNotFound, got %v", err)
	}

	// Verify expired key was deleted
	if _, ok := d.data["key1"]; ok {
		t.Error("Expire should delete expired key")
	}
}

func TestMemoryDriver_Persist_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	err := d.Persist(ctx, "key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Persist on expired key: expected ErrNotFound, got %v", err)
	}

	// Verify expired key was deleted
	if _, ok := d.data["key1"]; ok {
		t.Error("Persist should delete expired key")
	}
}

func TestMemoryDriver_Keys_NoPrefix(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "prefix:key1", []byte("value1"), 0)
	d.Set(ctx, "other:key2", []byte("value2"), 0)

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

func TestMemoryDriver_Incr_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	// Set expired key
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, 100)
	d.Set(ctx, "key1", buf, 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	// Incr on expired key should start from 0
	val, err := d.Incr(ctx, "key1", 5)
	if err != nil {
		t.Fatalf("Incr on expired key failed: %v", err)
	}

	if val != 5 {
		t.Errorf("Incr on expired key = %d, want 5", val)
	}
}

func TestMemoryDriver_GetSet_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("old"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, err := d.GetSet(ctx, "key1", []byte("new"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetSet on expired key: expected ErrNotFound, got %v", err)
	}

	// Should have set new value
	data, _ := d.Get(ctx, "key1")
	if string(data) != "new" {
		t.Errorf("GetSet should set new value = %q, want %q", data, "new")
	}
}

func TestMemoryDriver_CompareAndSwap_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("old"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	ok, err := d.CompareAndSwap(ctx, "key1", []byte("old"), []byte("new"), 0)
	if err != nil {
		t.Fatalf("CompareAndSwap on expired key failed: %v", err)
	}

	if ok {
		t.Error("CompareAndSwap on expired key should return false")
	}

	// Verify expired key was deleted
	if _, ok := d.data["key1"]; ok {
		t.Error("CompareAndSwap should delete expired key")
	}
}

func TestClient_MGet_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("driver error")

	mock := &mockDriver{
		mgetFunc: func(ctx context.Context, keys []string) (map[string][]byte, error) {
			return nil, expectedErr
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))
	_, err := c.MGet(context.Background(), "key1")

	if !errors.Is(err, expectedErr) {
		t.Errorf("MGet should propagate driver error, got %v", err)
	}
}

func TestClient_Keys_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("driver error")

	mock := &mockDriver{
		keysFunc: func(ctx context.Context, prefix, pattern string) ([]string, error) {
			return nil, expectedErr
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))
	_, err := c.Keys(context.Background(), "*")

	if !errors.Is(err, expectedErr) {
		t.Errorf("Keys should propagate driver error, got %v", err)
	}
}
