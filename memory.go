package namestore

import (
	"bytes"
	"context"
	"encoding/binary"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type entry struct {
	value  []byte
	expire time.Time
}

// Memory implements Driver with thread-safe in-memory storage.
type Memory struct {
	mu   sync.RWMutex
	data map[string]entry
}

// NewMemory creates an in-memory Driver instance.
func NewMemory() Driver {
	return &Memory{data: make(map[string]entry)}
}

// NewInMemoryDriver is an alias for NewMemory for backward compatibility.
// Deprecated: Use NewMemory instead.
func NewInMemoryDriver() Driver {
	return NewMemory()
}

func (m *Memory) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = entry{value: clone(value), expire: expiry(ttl)}
	return nil
}

func (m *Memory) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry, ok := m.data[key]; ok {
		if entry.expired() {
			delete(m.data, key)
		} else {
			return false, nil
		}
	}
	m.data[key] = entry{value: clone(value), expire: expiry(ttl)}
	return true, nil
}

func (m *Memory) Get(ctx context.Context, key string) ([]byte, error) {
	// Fast path: optimistic read with RLock.
	m.mu.RLock()
	e, ok := m.data[key]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	// Check expiration without lock first.
	if !e.expired() {
		return clone(e.value), nil
	}

	// Slow path: entry expired, need write lock to delete.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Re-check after acquiring write lock (double-check pattern).
	e, ok = m.data[key]
	if !ok {
		return nil, ErrNotFound
	}

	if e.expired() {
		delete(m.data, key)
		return nil, ErrNotFound
	}

	return clone(e.value), nil
}

func (m *Memory) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *Memory) Exists(ctx context.Context, key string) (bool, error) {
	// Fast path: optimistic read with RLock.
	m.mu.RLock()
	e, ok := m.data[key]
	m.mu.RUnlock()

	if !ok {
		return false, nil
	}

	// Check expiration without lock first.
	if !e.expired() {
		return true, nil
	}

	// Slow path: entry expired, need write lock to delete.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Re-check after acquiring write lock.
	e, ok = m.data[key]
	if !ok {
		return false, nil
	}

	if e.expired() {
		delete(m.data, key)
		return false, nil
	}

	return true, nil
}

func (e entry) expired() bool {
	if e.expire.IsZero() {
		return false
	}
	return time.Now().After(e.expire)
}

func expiry(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
}

func clone(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

// MGet retrieves multiple keys.
func (m *Memory) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string][]byte, len(keys))
	for _, key := range keys {
		if entry, ok := m.data[key]; ok && !entry.expired() {
			result[key] = clone(entry.value)
		}
	}

	return result, nil
}

// MSet sets multiple key-value pairs.
func (m *Memory) MSet(ctx context.Context, pairs map[string][]byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp := expiry(ttl)
	for key, value := range pairs {
		m.data[key] = entry{value: clone(value), expire: exp}
	}

	return nil
}

// MDel deletes multiple keys.
func (m *Memory) MDel(ctx context.Context, keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.data, key)
	}

	return nil
}

// TTL returns the remaining time-to-live. Returns -1 if key has no expiration, ErrNotFound if key doesn't exist.
func (m *Memory) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.data[key]
	if !ok || entry.expired() {
		if ok {
			delete(m.data, key)
		}
		return 0, ErrNotFound
	}

	if entry.expire.IsZero() {
		return -1, nil
	}

	return time.Until(entry.expire), nil
}

// Expire sets or updates the TTL for a key.
func (m *Memory) Expire(ctx context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.data[key]
	if !ok || entry.expired() {
		if ok {
			delete(m.data, key)
		}
		return ErrNotFound
	}

	entry.expire = expiry(ttl)
	m.data[key] = entry
	return nil
}

// Persist removes the expiration from a key.
func (m *Memory) Persist(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.data[key]
	if !ok || entry.expired() {
		if ok {
			delete(m.data, key)
		}
		return ErrNotFound
	}

	entry.expire = time.Time{}
	m.data[key] = entry
	return nil
}

// Keys returns all keys matching the prefix and pattern.
func (m *Memory) Keys(ctx context.Context, prefix, pattern string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []string
	for key := range m.data {
		if !strings.HasPrefix(key, prefix+":") {
			continue
		}

		if pattern != "" && pattern != "*" {
			matched, err := filepath.Match(pattern, key[len(prefix)+1:])
			if err != nil {
				return nil, ErrInvalidPattern
			}
			if !matched {
				continue
			}
		}

		if entry := m.data[key]; !entry.expired() {
			result = append(result, key)
		}
	}

	return result, nil
}

// Clear removes all keys with the given prefix.
func (m *Memory) Clear(ctx context.Context, prefix string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	keysToDelete := make([]string, 0)
	for key := range m.data {
		if strings.HasPrefix(key, prefix+":") {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(m.data, key)
	}

	return nil
}

// Incr atomically increments the integer value.
func (m *Memory) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.data[key]
	if ok && e.expired() {
		delete(m.data, key)
		ok = false
	}

	var current int64
	if ok {
		if len(e.value) != 8 {
			return 0, ErrTypeMismatch
		}
		current = int64(binary.LittleEndian.Uint64(e.value))
	}

	newValue := current + delta
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(newValue))

	if ok {
		e.value = buf
		m.data[key] = e
	} else {
		m.data[key] = entry{value: buf, expire: time.Time{}}
	}

	return newValue, nil
}

// Decr atomically decrements the integer value.
func (m *Memory) Decr(ctx context.Context, key string, delta int64) (int64, error) {
	return m.Incr(ctx, key, -delta)
}

// GetSet atomically sets a key to a new value and returns the old value.
func (m *Memory) GetSet(ctx context.Context, key string, value []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.data[key]
	if !ok || e.expired() {
		if ok {
			delete(m.data, key)
		}
		m.data[key] = entry{value: clone(value), expire: time.Time{}}
		return nil, ErrNotFound
	}

	oldValue := clone(e.value)
	e.value = clone(value)
	m.data[key] = e

	return oldValue, nil
}

// CompareAndSwap atomically compares and swaps if oldValue matches.
func (m *Memory) CompareAndSwap(ctx context.Context, key string, oldValue, newValue []byte, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.data[key]
	if !ok || e.expired() {
		if ok {
			delete(m.data, key)
		}
		return false, nil
	}

	if !bytes.Equal(e.value, oldValue) {
		return false, nil
	}

	m.data[key] = entry{value: clone(newValue), expire: expiry(ttl)}
	return true, nil
}
