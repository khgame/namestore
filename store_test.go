package namestore

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockDriver struct {
	setFunc    func(ctx context.Context, key string, value []byte, ttl time.Duration) error
	setNXFunc  func(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	getFunc    func(ctx context.Context, key string) ([]byte, error)
	deleteFunc func(ctx context.Context, key string) error
	existsFunc func(ctx context.Context, key string) (bool, error)
	mgetFunc   func(ctx context.Context, keys []string) (map[string][]byte, error)
	keysFunc   func(ctx context.Context, prefix, pattern string) ([]string, error)
}

func (m *mockDriver) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, value, ttl)
	}
	return nil
}

func (m *mockDriver) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	if m.setNXFunc != nil {
		return m.setNXFunc(ctx, key, value, ttl)
	}
	return true, nil
}

func (m *mockDriver) Get(ctx context.Context, key string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	return []byte("value"), nil
}

func (m *mockDriver) Delete(ctx context.Context, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key)
	}
	return nil
}

func (m *mockDriver) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, key)
	}
	return true, nil
}

// Stub implementations for extended Driver interface
func (m *mockDriver) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if m.mgetFunc != nil {
		return m.mgetFunc(ctx, keys)
	}
	return nil, nil
}

func (m *mockDriver) MSet(ctx context.Context, pairs map[string][]byte, ttl time.Duration) error {
	return nil
}

func (m *mockDriver) MDel(ctx context.Context, keys []string) error {
	return nil
}

func (m *mockDriver) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, ErrNotFound
}

func (m *mockDriver) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return ErrNotFound
}

func (m *mockDriver) Persist(ctx context.Context, key string) error {
	return ErrNotFound
}

func (m *mockDriver) Keys(ctx context.Context, prefix, pattern string) ([]string, error) {
	if m.keysFunc != nil {
		return m.keysFunc(ctx, prefix, pattern)
	}
	return nil, nil
}

func (m *mockDriver) Clear(ctx context.Context, prefix string) error {
	return nil
}

func (m *mockDriver) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, ErrNotFound
}

func (m *mockDriver) Decr(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, ErrNotFound
}

func (m *mockDriver) GetSet(ctx context.Context, key string, value []byte) ([]byte, error) {
	return nil, ErrNotFound
}

func (m *mockDriver) CompareAndSwap(ctx context.Context, key string, oldValue, newValue []byte, ttl time.Duration) (bool, error) {
	return false, nil
}

func TestWithDriver(t *testing.T) {
	mock := &mockDriver{}
	c := New[string]("root", "domain", WithDriver[string](mock))

	impl, ok := c.(*client[string])
	if !ok {
		t.Fatal("expected *client")
	}

	if impl.driver != mock {
		t.Errorf("WithDriver failed: expected mock driver")
	}
}

func TestWithDriver_NilIgnored(t *testing.T) {
	c := New[string]("root", "domain", WithDriver[string](nil))

	impl, ok := c.(*client[string])
	if !ok {
		t.Fatal("expected *client")
	}

	// Should keep default Memory when nil is passed
	if _, ok := impl.driver.(*Memory); !ok {
		t.Errorf("nil driver should keep default Memory, got %T", impl.driver)
	}
}

func TestNew_DefaultDriver(t *testing.T) {
	c := New[string]("root", "domain")

	impl, ok := c.(*client[string])
	if !ok {
		t.Fatal("expected *client")
	}

	// Should default to Memory
	if _, ok := impl.driver.(*Memory); !ok {
		t.Errorf("New should default to Memory, got %T", impl.driver)
	}
}

func TestClient_KeyConstruction(t *testing.T) {
	c := &client[string]{
		prefix: "root:domain",
		driver: &mockDriver{},
	}

	key := c.key("bizKey")
	expected := "root:domain:bizKey"
	if key != expected {
		t.Errorf("key() = %q, want %q", key, expected)
	}
}

func TestClient_Set(t *testing.T) {
	var capturedKey string
	var capturedValue []byte
	var capturedTTL time.Duration

	mock := &mockDriver{
		setFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) error {
			capturedKey = key
			capturedValue = value
			capturedTTL = ttl
			return nil
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))
	err := c.Set(context.Background(), "key1", []byte("data"), 10*time.Second)

	if err != nil {
		t.Errorf("Set returned error: %v", err)
	}

	if capturedKey != "root:domain:key1" {
		t.Errorf("Set key = %q, want %q", capturedKey, "root:domain:key1")
	}

	if string(capturedValue) != "data" {
		t.Errorf("Set value = %q, want %q", capturedValue, "data")
	}

	if capturedTTL != 10*time.Second {
		t.Errorf("Set ttl = %v, want %v", capturedTTL, 10*time.Second)
	}
}

func TestClient_SetNX(t *testing.T) {
	mock := &mockDriver{
		setNXFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
			if key == "root:domain:existing" {
				return false, nil
			}
			return true, nil
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))

	ok, err := c.SetNX(context.Background(), "new", []byte("data"), 10*time.Second)
	if err != nil {
		t.Errorf("SetNX returned error: %v", err)
	}
	if !ok {
		t.Error("SetNX should succeed for new key")
	}

	ok, err = c.SetNX(context.Background(), "existing", []byte("data"), 10*time.Second)
	if err != nil {
		t.Errorf("SetNX returned error: %v", err)
	}
	if ok {
		t.Error("SetNX should fail for existing key")
	}
}

func TestClient_Get(t *testing.T) {
	mock := &mockDriver{
		getFunc: func(ctx context.Context, key string) ([]byte, error) {
			if key == "root:domain:key1" {
				return []byte("value1"), nil
			}
			return nil, ErrNotFound
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))

	data, err := c.Get(context.Background(), "key1")
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if string(data) != "value1" {
		t.Errorf("Get returned %q, want %q", data, "value1")
	}

	_, err = c.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get missing key: expected ErrNotFound, got %v", err)
	}
}

func TestClient_Delete(t *testing.T) {
	var capturedKey string

	mock := &mockDriver{
		deleteFunc: func(ctx context.Context, key string) error {
			capturedKey = key
			return nil
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))
	err := c.Delete(context.Background(), "key1")

	if err != nil {
		t.Errorf("Delete returned error: %v", err)
	}

	if capturedKey != "root:domain:key1" {
		t.Errorf("Delete key = %q, want %q", capturedKey, "root:domain:key1")
	}
}

func TestClient_Exists(t *testing.T) {
	mock := &mockDriver{
		existsFunc: func(ctx context.Context, key string) (bool, error) {
			return key == "root:domain:existing", nil
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))

	exists, err := c.Exists(context.Background(), "existing")
	if err != nil {
		t.Errorf("Exists returned error: %v", err)
	}
	if !exists {
		t.Error("Exists should return true for existing key")
	}

	exists, err = c.Exists(context.Background(), "missing")
	if err != nil {
		t.Errorf("Exists returned error: %v", err)
	}
	if exists {
		t.Error("Exists should return false for missing key")
	}
}

type customKey string

func TestClient_CustomKeyType(t *testing.T) {
	c := New[customKey]("root", "domain")

	ctx := context.Background()
	key := customKey("custom")

	err := c.Set(ctx, key, []byte("value"), 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	data, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(data) != "value" {
		t.Errorf("Get returned %q, want %q", data, "value")
	}
}

func TestClient_DriverError(t *testing.T) {
	expectedErr := errors.New("driver error")

	mock := &mockDriver{
		setFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) error {
			return expectedErr
		},
	}

	c := New[string]("root", "domain", WithDriver[string](mock))
	err := c.Set(context.Background(), "key", []byte("value"), 0)

	if !errors.Is(err, expectedErr) {
		t.Errorf("Set should propagate driver error, got %v", err)
	}
}
