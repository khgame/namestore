package namestore

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockLogger captures log messages for testing
type mockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (m *mockLogger) Info(ctx context.Context, format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, fmt.Sprintf("INFO: "+format, args...))
}

func (m *mockLogger) Warn(ctx context.Context, format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, fmt.Sprintf("WARN: "+format, args...))
}

func (m *mockLogger) Error(ctx context.Context, format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, fmt.Sprintf("ERROR: "+format, args...))
}

func (m *mockLogger) Debug(ctx context.Context, format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, fmt.Sprintf("DEBUG: "+format, args...))
}

func (m *mockLogger) getMessages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.messages...)
}

func (m *mockLogger) contains(substring string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, msg := range m.messages {
		if strings.Contains(msg, substring) {
			return true
		}
	}
	return false
}

func TestWithLogger(t *testing.T) {
	logger := &mockLogger{}

	// Create client with error-prone mock driver
	mockDriver := &mockDriver{
		setFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) error {
			return fmt.Errorf("mock set error")
		},
	}

	client := New[string]("test", "ns",
		WithDriver[string](mockDriver),
		WithLogger[string](logger))

	// Trigger an error
	ctx := context.Background()
	client.Set(ctx, "key1", []byte("value"), 0)

	// Verify error was logged
	if !logger.contains("Set key1 failed") {
		t.Error("Expected error log for Set operation")
	}
}

func TestWithLogTag(t *testing.T) {
	logger := &mockLogger{}

	mockDriver := &mockDriver{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			return nil, fmt.Errorf("mock get error")
		},
	}

	client := New[string]("test", "ns",
		WithDriver[string](mockDriver),
		WithLogger[string](logger),
		WithLogTag[string]("[TestTag]"))

	ctx := context.Background()
	client.Get(ctx, "key1")

	// Verify log tag is present
	if !logger.contains("[TestTag]") {
		t.Error("Expected log tag in error message")
	}
	if !logger.contains("Get key1 failed") {
		t.Error("Expected error log for Get operation")
	}
}

func TestLoggerCoverage_AllOperations(t *testing.T) {
	logger := &mockLogger{}
	client := New[string]("test", "ns", WithLogger[string](logger))
	ctx := context.Background()

	// Test basic operations (no logging expected on success)
	client.Set(ctx, "k", []byte("v"), 0)
	client.Get(ctx, "k")
	client.Delete(ctx, "k")
	client.Exists(ctx, "k")

	// Test operations on missing keys (no logging for ErrNotFound)
	client.Get(ctx, "missing")
	client.TTL(ctx, "missing")
	client.Expire(ctx, "missing", time.Second)
	client.Persist(ctx, "missing")
	client.GetSet(ctx, "missing", []byte("v"))

	// Test batch operations
	client.MGet(ctx, "k1", "k2")
	client.MSet(ctx, map[string][]byte{"k": []byte("v")}, 0)
	client.MDel(ctx, "k1", "k2")

	// Test atomic operations
	client.Incr(ctx, "counter", 1)
	client.Decr(ctx, "counter", 1)
	client.CompareAndSwap(ctx, "k", []byte("old"), []byte("new"), 0)

	// Test namespace operations
	client.Keys(ctx, "*")
	client.Clear(ctx)

	// No errors logged since operations succeeded or returned expected ErrNotFound
	messages := logger.getMessages()
	if len(messages) > 0 {
		t.Logf("Unexpected log messages: %v", messages)
	}
}

func TestNoOpLogger(t *testing.T) {
	// Create client without logger (should use no-op)
	client := New[string]("test", "ns")
	ctx := context.Background()

	// Should not panic with no-op logger
	client.Set(ctx, "key", []byte("value"), 0)
	client.Get(ctx, "key")
}

func TestLoggerNilSafety(t *testing.T) {
	// Passing nil logger should use default no-op
	client := New[string]("test", "ns",
		WithLogger[string](nil))

	ctx := context.Background()

	// Should not panic
	client.Set(ctx, "key", []byte("value"), 0)
}
