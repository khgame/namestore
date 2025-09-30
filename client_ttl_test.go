package namestore

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestClient_TTL tests retrieving the time-to-live for keys.
func TestClient_TTL(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Key with TTL.
	_ = c.Set(ctx, "key1", []byte("value1"), 100*time.Millisecond)
	ttl, err := c.TTL(ctx, "key1")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl <= 0 || ttl > 100*time.Millisecond {
		t.Errorf("TTL = %v, expected > 0 and <= 100ms", ttl)
	}

	// Key without TTL.
	_ = c.Set(ctx, "key2", []byte("value2"), 0)
	ttl, err = c.TTL(ctx, "key2")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	if ttl != -1 {
		t.Errorf("TTL for key without expiration = %v, want -1", ttl)
	}

	// Missing key.
	_, err = c.TTL(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("TTL for missing key: expected ErrNotFound, got %v", err)
	}
}

// TestClient_Expire tests adding expiration to existing keys.
func TestClient_Expire(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set key without TTL.
	_ = c.Set(ctx, "key1", []byte("value1"), 0)

	// Add expiration.
	err := c.Expire(ctx, "key1", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Expire failed: %v", err)
	}

	// Verify key exists.
	exists, _ := c.Exists(ctx, "key1")
	if !exists {
		t.Error("key1 should exist after Expire")
	}

	// Wait for expiration.
	time.Sleep(20 * time.Millisecond)

	exists, _ = c.Exists(ctx, "key1")
	if exists {
		t.Error("key1 should expire after TTL")
	}
}

// TestClient_Expire_MissingKey tests Expire on a non-existent key.
func TestClient_Expire_MissingKey(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	err := c.Expire(ctx, "missing", 10*time.Second)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expire for missing key: expected ErrNotFound, got %v", err)
	}
}

// TestClient_Persist tests removing expiration from keys.
func TestClient_Persist(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	// Set key with TTL.
	_ = c.Set(ctx, "key1", []byte("value1"), 10*time.Millisecond)

	// Remove expiration.
	err := c.Persist(ctx, "key1")
	if err != nil {
		t.Fatalf("Persist failed: %v", err)
	}

	// Wait longer than original TTL.
	time.Sleep(20 * time.Millisecond)

	// Key should still exist.
	exists, _ := c.Exists(ctx, "key1")
	if !exists {
		t.Error("key1 should still exist after Persist")
	}

	// Verify TTL is -1.
	ttl, _ := c.TTL(ctx, "key1")
	if ttl != -1 {
		t.Errorf("TTL after Persist = %v, want -1", ttl)
	}
}

// TestClient_Persist_MissingKey tests Persist on a non-existent key.
func TestClient_Persist_MissingKey(t *testing.T) {
	c := New[string]("root", "domain")
	ctx := context.Background()

	err := c.Persist(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Persist for missing key: expected ErrNotFound, got %v", err)
	}
}
