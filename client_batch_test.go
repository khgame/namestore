package namestore

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"
)

// TestClient_MGet tests batch retrieval of multiple keys.
func TestClient_MGet(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data.
	_ = c.Set(ctx, "key1", []byte("value1"), 0)
	_ = c.Set(ctx, "key2", []byte("value2"), 0)

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

// TestClient_MGet_Empty tests MGet with no keys.
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

// TestClient_MGet_ErrorPropagation tests that MGet propagates driver errors.
func TestClient_MGet_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("driver error")

	mock := &mockDriver{
		mgetFunc: func(_ context.Context, _ []string) (map[string][]byte, error) {
			return nil, expectedErr
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))
	_, err := c.MGet(context.Background(), "key1")

	if !errors.Is(err, expectedErr) {
		t.Errorf("MGet should propagate driver error, got %v", err)
	}
}

// TestClient_MSet tests batch setting of multiple key-value pairs.
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

	// Verify all keys were set.
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

// TestClient_MSet_WithTTL tests MSet with TTL expiration.
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

	// Verify keys exist.
	exists, _ := c.Exists(ctx, "key1")
	if !exists {
		t.Error("key1 should exist after MSet")
	}

	// Wait for expiration.
	time.Sleep(20 * time.Millisecond)

	exists, _ = c.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should expire after TTL")
	}
}

// TestClient_MSet_Empty tests MSet with empty pairs.
func TestClient_MSet_Empty(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	err := c.MSet(ctx, map[string][]byte{}, 0)
	if err != nil {
		t.Fatalf("MSet with empty pairs failed: %v", err)
	}
}

// TestClient_MDel tests batch deletion of multiple keys.
func TestClient_MDel(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data.
	_ = c.Set(ctx, "key1", []byte("value1"), 0)
	_ = c.Set(ctx, "key2", []byte("value2"), 0)
	_ = c.Set(ctx, "key3", []byte("value3"), 0)

	err := c.MDel(ctx, "key1", "key2")
	if err != nil {
		t.Fatalf("MDel failed: %v", err)
	}

	// Verify deletion.
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

// TestClient_MDel_Empty tests MDel with no keys.
func TestClient_MDel_Empty(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	err := c.MDel(ctx)
	if err != nil {
		t.Fatalf("MDel with empty keys failed: %v", err)
	}
}

// TestClient_Keys tests retrieving all keys in a namespace.
func TestClient_Keys(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data.
	_ = c.Set(ctx, "user:1", []byte("alice"), 0)
	_ = c.Set(ctx, "user:2", []byte("bob"), 0)
	_ = c.Set(ctx, "admin:1", []byte("charlie"), 0)

	// Get all keys.
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

// TestClient_Keys_WithPattern tests Keys with pattern matching.
func TestClient_Keys_WithPattern(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data.
	_ = c.Set(ctx, "user:1", []byte("alice"), 0)
	_ = c.Set(ctx, "user:2", []byte("bob"), 0)
	_ = c.Set(ctx, "admin:1", []byte("charlie"), 0)

	// Get keys matching pattern.
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

// TestClient_Keys_Empty tests Keys on an empty namespace.
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

// TestClient_Keys_ErrorPropagation tests that Keys propagates driver errors.
func TestClient_Keys_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("driver error")

	mock := &mockDriver{
		keysFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return nil, expectedErr
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))
	_, err := c.Keys(context.Background(), "*")

	if !errors.Is(err, expectedErr) {
		t.Errorf("Keys should propagate driver error, got %v", err)
	}
}

// TestClient_Clear tests clearing all keys in a namespace.
func TestClient_Clear(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set up test data.
	_ = c.Set(ctx, "key1", []byte("value1"), 0)
	_ = c.Set(ctx, "key2", []byte("value2"), 0)

	err := c.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify all keys are deleted.
	exists, _ := c.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should be cleared")
	}

	exists, _ = c.Exists(ctx, "key2")
	if exists {
		t.Error("key2 should be cleared")
	}
}

// TestClient_Clear_IsolatesNamespaces tests that Clear only affects the specified namespace.
func TestClient_Clear_IsolatesNamespaces(t *testing.T) {
	c1 := New[string]("root", "domain1")
	c2 := New[string]("root", "domain2")
	ctx := context.Background()

	// Set up test data in both namespaces.
	_ = c1.Set(ctx, "key1", []byte("value1"), 0)
	_ = c2.Set(ctx, "key1", []byte("value2"), 0)

	// Clear only domain1.
	_ = c1.Clear(ctx)

	// domain1 should be empty.
	exists, _ := c1.Exists(ctx, "key1")
	if exists {
		t.Error("domain1 key1 should be cleared")
	}

	// domain2 should be untouched.
	exists, _ = c2.Exists(ctx, "key1")
	if !exists {
		t.Error("domain2 key1 should still exist")
	}
}
