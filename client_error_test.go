package namestore

import (
	"context"
	"errors"
	"testing"
	"time"
)

var (
	errMockSet     = errors.New("mock set error")
	errMockSetNX   = errors.New("mock setNX error")
	errMockGet     = errors.New("mock get error")
	errMockDelete  = errors.New("mock delete error")
	errMockExists  = errors.New("mock exists error")
	errMockMSet    = errors.New("mock mset error")
	errMockMDel    = errors.New("mock mdel error")
	errMockTTL     = errors.New("mock ttl error")
	errMockExpire  = errors.New("mock expire error")
	errMockPersist = errors.New("mock persist error")
	errMockKeys    = errors.New("mock keys error")
	errMockClear   = errors.New("mock clear error")
	errMockIncr    = errors.New("mock incr error")
	errMockDecr    = errors.New("mock decr error")
	errMockGetSet  = errors.New("mock getset error")
	errMockCAS     = errors.New("mock cas error")
)

// errorDriver is a mock driver that always returns errors.
type errorDriver struct {
	Memory
}

func (*errorDriver) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return errMockSet
}

func (*errorDriver) SetNX(_ context.Context, _ string, _ []byte, _ time.Duration) (bool, error) {
	return false, errMockSetNX
}

func (*errorDriver) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, errMockGet
}

func (*errorDriver) Delete(_ context.Context, _ string) error {
	return errMockDelete
}

func (*errorDriver) Exists(_ context.Context, _ string) (bool, error) {
	return false, errMockExists
}

func (*errorDriver) MSet(_ context.Context, _ map[string][]byte, _ time.Duration) error {
	return errMockMSet
}

func (*errorDriver) MDel(_ context.Context, _ []string) error {
	return errMockMDel
}

func (*errorDriver) TTL(_ context.Context, _ string) (time.Duration, error) {
	return 0, errMockTTL
}

func (*errorDriver) Expire(_ context.Context, _ string, _ time.Duration) error {
	return errMockExpire
}

func (*errorDriver) Persist(_ context.Context, _ string) error {
	return errMockPersist
}

func (*errorDriver) Keys(_ context.Context, _, _ string) ([]string, error) {
	return nil, errMockKeys
}

func (*errorDriver) Clear(_ context.Context, _ string) error {
	return errMockClear
}

func (*errorDriver) Incr(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, errMockIncr
}

func (*errorDriver) Decr(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, errMockDecr
}

func (*errorDriver) GetSet(_ context.Context, _ string, _ []byte) ([]byte, error) {
	return nil, errMockGetSet
}

func (*errorDriver) CompareAndSwap(_ context.Context, _ string, _, _ []byte, _ time.Duration) (bool, error) {
	return false, errMockCAS
}

// TestErrorPaths_SetNX tests SetNX error path.
func TestErrorPaths_SetNX(t *testing.T) {
	logger := &mockLogger{}
	driver := &errorDriver{}
	client := New[string]("test", "ns",
		WithDriver[string](driver),
		WithLogger[string](logger))

	ctx := context.Background()
	_, err := client.SetNX(ctx, "key", []byte("value"), 0)
	if err == nil {
		t.Error("Expected error from SetNX")
	}
	if !logger.contains("SetNX") {
		t.Error("Expected SetNX error to be logged")
	}
}

// TestErrorPaths_Delete tests Delete error path.
func TestErrorPaths_Delete(t *testing.T) {
	logger := &mockLogger{}
	driver := &errorDriver{}
	client := New[string]("test", "ns",
		WithDriver[string](driver),
		WithLogger[string](logger))

	ctx := context.Background()
	err := client.Delete(ctx, "key")
	if err == nil {
		t.Error("Expected error from Delete")
	}
	if !logger.contains("Delete") {
		t.Error("Expected Delete error to be logged")
	}
}

// TestErrorPaths_Exists tests Exists error path.
func TestErrorPaths_Exists(t *testing.T) {
	logger := &mockLogger{}
	driver := &errorDriver{}
	client := New[string]("test", "ns",
		WithDriver[string](driver),
		WithLogger[string](logger))

	ctx := context.Background()
	_, err := client.Exists(ctx, "key")
	if err == nil {
		t.Error("Expected error from Exists")
	}
	if !logger.contains("Exists") {
		t.Error("Expected Exists error to be logged")
	}
}

// TestErrorPaths_Batch tests batch operation error paths.
func TestErrorPaths_Batch(t *testing.T) {
	logger := &mockLogger{}
	driver := &errorDriver{}
	client := New[string]("test", "ns",
		WithDriver[string](driver),
		WithLogger[string](logger))

	ctx := context.Background()

	err := client.MSet(ctx, map[string][]byte{"key": []byte("value")}, 0)
	if err == nil || !logger.contains("MSet") {
		t.Error("Expected MSet error to be logged")
	}

	err = client.MDel(ctx, "key1", "key2")
	if err == nil || !logger.contains("MDel") {
		t.Error("Expected MDel error to be logged")
	}
}

// TestErrorPaths_TTL tests TTL management error paths.
func TestErrorPaths_TTL(t *testing.T) {
	logger := &mockLogger{}
	driver := &errorDriver{}
	client := New[string]("test", "ns",
		WithDriver[string](driver),
		WithLogger[string](logger))

	ctx := context.Background()

	_, err := client.TTL(ctx, "key")
	if err == nil || !logger.contains("TTL") {
		t.Error("Expected TTL error to be logged")
	}

	err = client.Expire(ctx, "key", time.Second)
	if err == nil || !logger.contains("Expire") {
		t.Error("Expected Expire error to be logged")
	}

	err = client.Persist(ctx, "key")
	if err == nil || !logger.contains("Persist") {
		t.Error("Expected Persist error to be logged")
	}
}

// TestErrorPaths_Namespace tests namespace operation error paths.
func TestErrorPaths_Namespace(t *testing.T) {
	logger := &mockLogger{}
	driver := &errorDriver{}
	client := New[string]("test", "ns",
		WithDriver[string](driver),
		WithLogger[string](logger))

	ctx := context.Background()

	err := client.Clear(ctx)
	if err == nil || !logger.contains("Clear") {
		t.Error("Expected Clear error to be logged")
	}
}

// TestErrorPaths_Atomic tests atomic operation error paths.
func TestErrorPaths_Atomic(t *testing.T) {
	logger := &mockLogger{}
	driver := &errorDriver{}
	client := New[string]("test", "ns",
		WithDriver[string](driver),
		WithLogger[string](logger))

	ctx := context.Background()

	_, err := client.Decr(ctx, "key", 1)
	if err == nil || !logger.contains("Decr") {
		t.Error("Expected Decr error to be logged")
	}

	_, err = client.GetSet(ctx, "key", []byte("value"))
	if err == nil || !logger.contains("GetSet") {
		t.Error("Expected GetSet error to be logged")
	}

	_, err = client.CompareAndSwap(ctx, "key", []byte("old"), []byte("new"), 0)
	if err == nil || !logger.contains("CompareAndSwap") {
		t.Error("Expected CompareAndSwap error to be logged")
	}
}

// TestLogfAllLevels tests all log levels to cover switch branches.
func TestLogfAllLevels(t *testing.T) {
	logger := &mockLogger{}
	c := New[string]("test", "ns",
		WithLogger[string](logger),
		WithLogTag[string]("[TestTag]"))

	ctx := context.Background()

	// Access the implementation to test logf directly.
	type clientImpl interface {
		logf(level string, ctx context.Context, format string, args ...any)
	}

	impl, ok := c.(clientImpl)
	if !ok {
		t.Fatal("Failed to cast to implementation")
	}

	// Test all log levels.
	impl.logf("info", ctx, "info message")
	impl.logf("warn", ctx, "warn message")
	impl.logf("error", ctx, "error message")
	impl.logf("debug", ctx, "debug message")

	messages := logger.getMessages()
	if len(messages) != 4 {
		t.Errorf("Expected 4 log messages, got %d", len(messages))
	}

	// Verify all levels were logged.
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	for _, level := range levels {
		found := false
		for _, msg := range messages {
			if contains := func(s, substr string) bool {
				return len(s) >= len(substr) && s[:len(substr)] == substr
			}(msg, level); contains {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s level message", level)
		}
	}

	// Verify tag is present.
	if !logger.contains("[TestTag]") {
		t.Error("Expected log tag in messages")
	}
}

// TestNoOpLoggerMethods tests the noopLogger methods are callable.
func TestNoOpLoggerMethods(_ *testing.T) {
	logger := noopLogger{}
	ctx := context.Background()

	// These should not panic.
	logger.Info(ctx, "test")
	logger.Warn(ctx, "test")
	logger.Error(ctx, "test")
	logger.Debug(ctx, "test")
}