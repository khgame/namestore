package namestore

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryDriver(t *testing.T) {
	d := NewInMemoryDriver()
	if d == nil {
		t.Fatal("NewInMemoryDriver returned nil")
	}

	md, ok := d.(*Memory)
	if !ok {
		t.Fatalf("NewInMemoryDriver returned %T, want *Memory", d)
	}

	if md.data == nil {
		t.Error("NewInMemoryDriver did not initialize data map")
	}
}

func TestMemoryDriver_Set(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	err := d.Set(ctx, "key1", []byte("value1"), 0)
	if err != nil {
		t.Errorf("Set returned error: %v", err)
	}

	entry, ok := d.data["key1"]
	if !ok {
		t.Fatal("Set did not store key")
	}

	if string(entry.value) != "value1" {
		t.Errorf("Set stored %q, want %q", entry.value, "value1")
	}

	if !entry.expire.IsZero() {
		t.Error("Set with ttl=0 should have zero expire time")
	}
}

func TestMemoryDriver_Set_WithTTL(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	before := time.Now()
	err := d.Set(ctx, "key1", []byte("value1"), 100*time.Millisecond)
	after := time.Now()

	if err != nil {
		t.Errorf("Set returned error: %v", err)
	}

	entry := d.data["key1"]
	if entry.expire.IsZero() {
		t.Error("Set with ttl>0 should have non-zero expire time")
	}

	expectedMin := before.Add(100 * time.Millisecond)
	expectedMax := after.Add(100 * time.Millisecond)

	if entry.expire.Before(expectedMin) || entry.expire.After(expectedMax) {
		t.Errorf("expire time %v not in range [%v, %v]", entry.expire, expectedMin, expectedMax)
	}
}

func TestMemoryDriver_Set_Clone(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	original := []byte("value")
	err := d.Set(ctx, "key1", original, 0)
	if err != nil {
		t.Errorf("Set returned error: %v", err)
	}

	// Modify original
	original[0] = 'X'

	entry := d.data["key1"]
	if string(entry.value) != "value" {
		t.Errorf("Set did not clone value: got %q", entry.value)
	}
}

func TestMemoryDriver_SetNX_NewKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	ok, err := d.SetNX(ctx, "key1", []byte("value1"), 0)
	if err != nil {
		t.Errorf("SetNX returned error: %v", err)
	}
	if !ok {
		t.Error("SetNX should succeed for new key")
	}

	entry := d.data["key1"]
	if string(entry.value) != "value1" {
		t.Errorf("SetNX stored %q, want %q", entry.value, "value1")
	}
}

func TestMemoryDriver_SetNX_ExistingKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 0)

	ok, err := d.SetNX(ctx, "key1", []byte("value2"), 0)
	if err != nil {
		t.Errorf("SetNX returned error: %v", err)
	}
	if ok {
		t.Error("SetNX should fail for existing key")
	}

	entry := d.data["key1"]
	if string(entry.value) != "value1" {
		t.Errorf("SetNX should not overwrite existing key: got %q", entry.value)
	}
}

func TestMemoryDriver_SetNX_ExpiredKey(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	ok, err := d.SetNX(ctx, "key1", []byte("value2"), 0)
	if err != nil {
		t.Errorf("SetNX returned error: %v", err)
	}
	if !ok {
		t.Error("SetNX should succeed for expired key")
	}

	entry := d.data["key1"]
	if string(entry.value) != "value2" {
		t.Errorf("SetNX stored %q, want %q", entry.value, "value2")
	}
}

func TestMemoryDriver_Get(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 0)

	data, err := d.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}

	if string(data) != "value1" {
		t.Errorf("Get returned %q, want %q", data, "value1")
	}
}

func TestMemoryDriver_Get_NotFound(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	_, err := d.Get(ctx, "missing")
	if err != ErrNotFound {
		t.Errorf("Get missing key: expected ErrNotFound, got %v", err)
	}
}

func TestMemoryDriver_Get_Expired(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, err := d.Get(ctx, "key1")
	if err != ErrNotFound {
		t.Errorf("Get expired key: expected ErrNotFound, got %v", err)
	}

	// Verify expired key was deleted
	if _, ok := d.data["key1"]; ok {
		t.Error("Get should delete expired key")
	}
}

func TestMemoryDriver_Get_Clone(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value"), 0)

	data, err := d.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}

	// Modify returned data
	data[0] = 'X'

	// Verify original is unchanged
	data2, _ := d.Get(ctx, "key1")
	if string(data2) != "value" {
		t.Errorf("Get did not clone value: got %q", data2)
	}
}

func TestMemoryDriver_Delete(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 0)

	err := d.Delete(ctx, "key1")
	if err != nil {
		t.Errorf("Delete returned error: %v", err)
	}

	if _, ok := d.data["key1"]; ok {
		t.Error("Delete did not remove key")
	}
}

func TestMemoryDriver_Delete_NonExistent(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	err := d.Delete(ctx, "missing")
	if err != nil {
		t.Errorf("Delete non-existent key returned error: %v", err)
	}
}

func TestMemoryDriver_Exists(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 0)

	exists, err := d.Exists(ctx, "key1")
	if err != nil {
		t.Errorf("Exists returned error: %v", err)
	}
	if !exists {
		t.Error("Exists should return true for existing key")
	}
}

func TestMemoryDriver_Exists_NotFound(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	exists, err := d.Exists(ctx, "missing")
	if err != nil {
		t.Errorf("Exists returned error: %v", err)
	}
	if exists {
		t.Error("Exists should return false for missing key")
	}
}

func TestMemoryDriver_Exists_Expired(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	d.Set(ctx, "key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	exists, err := d.Exists(ctx, "key1")
	if err != nil {
		t.Errorf("Exists returned error: %v", err)
	}
	if exists {
		t.Error("Exists should return false for expired key")
	}

	// Verify expired key was deleted
	if _, ok := d.data["key1"]; ok {
		t.Error("Exists should delete expired key")
	}
}

func TestMemEntry_Expired(t *testing.T) {
	tests := []struct {
		name   string
		entry  entry
		expect bool
	}{
		{
			name:   "zero time never expires",
			entry:  entry{expire: time.Time{}},
			expect: false,
		},
		{
			name:   "future time not expired",
			entry:  entry{expire: time.Now().Add(1 * time.Hour)},
			expect: false,
		},
		{
			name:   "past time expired",
			entry:  entry{expire: time.Now().Add(-1 * time.Hour)},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.expired()
			if result != tt.expect {
				t.Errorf("expired() = %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestExpiry(t *testing.T) {
	tests := []struct {
		name    string
		ttl     time.Duration
		isZero  bool
		minTime time.Time
		maxTime time.Time
	}{
		{
			name:   "zero ttl",
			ttl:    0,
			isZero: true,
		},
		{
			name:   "negative ttl",
			ttl:    -1 * time.Second,
			isZero: true,
		},
		{
			name:    "positive ttl",
			ttl:     100 * time.Millisecond,
			isZero:  false,
			minTime: time.Now().Add(100 * time.Millisecond),
			maxTime: time.Now().Add(100 * time.Millisecond),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			result := expiry(tt.ttl)
			after := time.Now()

			if tt.isZero {
				if !result.IsZero() {
					t.Errorf("expiry(%v) should return zero time, got %v", tt.ttl, result)
				}
			} else {
				expectedMin := before.Add(tt.ttl)
				expectedMax := after.Add(tt.ttl)

				if result.Before(expectedMin) || result.After(expectedMax) {
					t.Errorf("expiry(%v) = %v, expected in range [%v, %v]", tt.ttl, result, expectedMin, expectedMax)
				}
			}
		})
	}
}

func TestClone(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		expect []byte
	}{
		{
			name:   "nil slice",
			input:  nil,
			expect: nil,
		},
		{
			name:   "empty slice",
			input:  []byte{},
			expect: nil,
		},
		{
			name:   "non-empty slice",
			input:  []byte("hello"),
			expect: []byte("hello"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clone(tt.input)

			if !reflect.DeepEqual(result, tt.expect) {
				t.Errorf("clone(%v) = %v, want %v", tt.input, result, tt.expect)
			}

			// Verify independence
			if len(tt.input) > 0 && len(result) > 0 {
				result[0] = 'X'
				if tt.input[0] == 'X' {
					t.Error("clone did not create independent copy")
				}
			}
		})
	}
}

func TestMemoryDriver_Concurrency(t *testing.T) {
	d := NewInMemoryDriver()
	ctx := context.Background()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			value := []byte{byte(n)}
			d.Set(ctx, key, value, 0)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Get(ctx, "key")
		}()
	}

	// Concurrent deletes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Delete(ctx, "key")
		}()
	}

	// Concurrent exists
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Exists(ctx, "key")
		}()
	}

	wg.Wait()
}

func TestMemoryDriver_SetNX_Concurrency(t *testing.T) {
	d := NewInMemoryDriver()
	ctx := context.Background()

	var wg sync.WaitGroup
	iterations := 100
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ok, _ := d.SetNX(ctx, "key", []byte{byte(n)}, 0)
			if ok {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Errorf("SetNX concurrency test: expected exactly 1 success, got %d", successCount)
	}
}
