package namestore

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound       = errors.New("namestore: not found")
	ErrTypeMismatch   = errors.New("namestore: type mismatch")
	ErrInvalidPattern = errors.New("namestore: invalid pattern")
)

// Driver describes comprehensive KV storage operations.
// Implementations must be thread-safe.
type Driver interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Batch operations
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	MSet(ctx context.Context, pairs map[string][]byte, ttl time.Duration) error
	MDel(ctx context.Context, keys []string) error

	// TTL management
	TTL(ctx context.Context, key string) (time.Duration, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Persist(ctx context.Context, key string) error

	// Namespace operations
	Keys(ctx context.Context, prefix, pattern string) ([]string, error)
	Clear(ctx context.Context, prefix string) error

	// Atomic operations
	Incr(ctx context.Context, key string, delta int64) (int64, error)
	Decr(ctx context.Context, key string, delta int64) (int64, error)
	GetSet(ctx context.Context, key string, value []byte) ([]byte, error)
	CompareAndSwap(ctx context.Context, key string, oldValue, newValue []byte, ttl time.Duration) (bool, error)
}

// Option customizes Client behavior.
type Option[TKey ~string] func(*client[TKey])

// WithDriver specifies the storage driver.
// If not provided, NewInMemoryDriver() will be used.
func WithDriver[TKey ~string](d Driver) Option[TKey] {
	return func(c *client[TKey]) {
		if d != nil {
			c.driver = d
		}
	}
}

// WithLogger specifies a logger for operation logging.
// If not provided, a no-op logger is used (no logging).
func WithLogger[TKey ~string](logger Logger) Option[TKey] {
	return func(c *client[TKey]) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithLogTag sets a tag prefix for all log messages.
// Useful for identifying the source of logs in multi-client scenarios.
func WithLogTag[TKey ~string](tag string) Option[TKey] {
	return func(c *client[TKey]) {
		c.logTag = tag
	}
}

// Client exposes namespaced KV storage operations.
// Keys are automatically prefixed with "rootNS:domain:".
type Client[TKey ~string] interface {
	Set(ctx context.Context, key TKey, value []byte, ttl time.Duration) error
	SetNX(ctx context.Context, key TKey, value []byte, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key TKey) ([]byte, error)
	Delete(ctx context.Context, key TKey) error
	Exists(ctx context.Context, key TKey) (bool, error)

	// Batch operations
	MGet(ctx context.Context, keys ...TKey) (map[TKey][]byte, error)
	MSet(ctx context.Context, pairs map[TKey][]byte, ttl time.Duration) error
	MDel(ctx context.Context, keys ...TKey) error

	// TTL management
	TTL(ctx context.Context, key TKey) (time.Duration, error)
	Expire(ctx context.Context, key TKey, ttl time.Duration) error
	Persist(ctx context.Context, key TKey) error

	// Namespace operations
	Keys(ctx context.Context, pattern string) ([]TKey, error)
	Clear(ctx context.Context) error

	// Atomic operations
	Incr(ctx context.Context, key TKey, delta int64) (int64, error)
	Decr(ctx context.Context, key TKey, delta int64) (int64, error)
	GetSet(ctx context.Context, key TKey, newValue []byte) ([]byte, error)
	CompareAndSwap(ctx context.Context, key TKey, oldValue, newValue []byte, ttl time.Duration) (bool, error)
}

type client[TKey ~string] struct {
	prefix          string
	prefixWithColon string
	driver          Driver
	logger          Logger
	logTag          string
}

// New creates a namespace-scoped Client.
// Keys are stored as "rootNS:domain:businessKey".
// If no driver is provided via WithDriver, NewInMemoryDriver() is used.
// If no logger is provided via WithLogger, a no-op logger is used (no logging).
func New[TKey ~string](rootNS, domain string, opts ...Option[TKey]) Client[TKey] {
	prefix := rootNS + ":" + domain
	c := &client[TKey]{
		prefix:          prefix,
		prefixWithColon: prefix + ":",
		driver:          NewInMemoryDriver(), // Default to in-memory
		logger:          defaultLogger,       // Default to no-op
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *client[TKey]) key(k TKey) string {
	if c.prefixWithColon != "" {
		return c.prefixWithColon + string(k)
	}
	if c.prefix == "" {
		return ":" + string(k)
	}
	return c.prefix + ":" + string(k)
}

func (c *client[TKey]) logf(level string, ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if c.logTag != "" {
		msg = c.logTag + " " + msg
	}
	switch level {
	case "info":
		c.logger.Info(ctx, msg)
	case "warn":
		c.logger.Warn(ctx, msg)
	case "error":
		c.logger.Error(ctx, msg)
	case "debug":
		c.logger.Debug(ctx, msg)
	}
}

func (c *client[TKey]) Set(ctx context.Context, key TKey, value []byte, ttl time.Duration) error {
	err := c.driver.Set(ctx, c.key(key), value, ttl)
	if err != nil {
		c.logf("error", ctx, "Set %s failed: %v", key, err)
	}
	return err
}

func (c *client[TKey]) SetNX(ctx context.Context, key TKey, value []byte, ttl time.Duration) (bool, error) {
	ok, err := c.driver.SetNX(ctx, c.key(key), value, ttl)
	if err != nil {
		c.logf("error", ctx, "SetNX %s failed: %v", key, err)
	}
	return ok, err
}

func (c *client[TKey]) Get(ctx context.Context, key TKey) ([]byte, error) {
	data, err := c.driver.Get(ctx, c.key(key))
	if err != nil && !errors.Is(err, ErrNotFound) {
		c.logf("error", ctx, "Get %s failed: %v", key, err)
	}
	return data, err
}

func (c *client[TKey]) Delete(ctx context.Context, key TKey) error {
	err := c.driver.Delete(ctx, c.key(key))
	if err != nil {
		c.logf("error", ctx, "Delete %s failed: %v", key, err)
	}
	return err
}

func (c *client[TKey]) Exists(ctx context.Context, key TKey) (bool, error) {
	exists, err := c.driver.Exists(ctx, c.key(key))
	if err != nil {
		c.logf("error", ctx, "Exists %s failed: %v", key, err)
	}
	return exists, err
}

// MGet retrieves multiple keys in a single call.
func (c *client[TKey]) MGet(ctx context.Context, keys ...TKey) (map[TKey][]byte, error) {
	if len(keys) == 0 {
		return make(map[TKey][]byte), nil
	}

	fullKeys := make([]string, len(keys))
	for i, k := range keys {
		fullKeys[i] = c.key(k)
	}

	result, err := c.driver.MGet(ctx, fullKeys)
	if err != nil {
		c.logf("error", ctx, "MGet failed: %v", err)
		return nil, err
	}

	// Convert back to business keys
	businessResult := make(map[TKey][]byte, len(result))
	for i, k := range keys {
		if data, ok := result[fullKeys[i]]; ok {
			businessResult[k] = data
		}
	}

	return businessResult, nil
}

// MSet sets multiple key-value pairs with the same TTL.
func (c *client[TKey]) MSet(ctx context.Context, pairs map[TKey][]byte, ttl time.Duration) error {
	if len(pairs) == 0 {
		return nil
	}

	fullPairs := make(map[string][]byte, len(pairs))
	for k, v := range pairs {
		fullPairs[c.key(k)] = v
	}

	err := c.driver.MSet(ctx, fullPairs, ttl)
	if err != nil {
		c.logf("error", ctx, "MSet failed: %v", err)
	}
	return err
}

// MDel deletes multiple keys in a single call.
func (c *client[TKey]) MDel(ctx context.Context, keys ...TKey) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, k := range keys {
		fullKeys[i] = c.key(k)
	}

	err := c.driver.MDel(ctx, fullKeys)
	if err != nil {
		c.logf("error", ctx, "MDel failed: %v", err)
	}
	return err
}

// TTL returns the remaining time-to-live for a key. Returns -1 if key has no expiration.
func (c *client[TKey]) TTL(ctx context.Context, key TKey) (time.Duration, error) {
	ttl, err := c.driver.TTL(ctx, c.key(key))
	if err != nil && !errors.Is(err, ErrNotFound) {
		c.logf("error", ctx, "TTL %s failed: %v", key, err)
	}
	return ttl, err
}

// Expire sets or updates the TTL for an existing key.
func (c *client[TKey]) Expire(ctx context.Context, key TKey, ttl time.Duration) error {
	err := c.driver.Expire(ctx, c.key(key), ttl)
	if err != nil && !errors.Is(err, ErrNotFound) {
		c.logf("error", ctx, "Expire %s failed: %v", key, err)
	}
	return err
}

// Persist removes the expiration from a key.
func (c *client[TKey]) Persist(ctx context.Context, key TKey) error {
	err := c.driver.Persist(ctx, c.key(key))
	if err != nil && !errors.Is(err, ErrNotFound) {
		c.logf("error", ctx, "Persist %s failed: %v", key, err)
	}
	return err
}

// Keys returns all business keys matching the pattern within this namespace.
func (c *client[TKey]) Keys(ctx context.Context, pattern string) ([]TKey, error) {
	fullKeys, err := c.driver.Keys(ctx, c.prefix, pattern)
	if err != nil {
		c.logf("error", ctx, "Keys pattern=%s failed: %v", pattern, err)
		return nil, err
	}

	// Strip prefix to get business keys
	prefixLen := len(c.prefix) + 1 // +1 for the colon
	businessKeys := make([]TKey, 0, len(fullKeys))
	for _, fullKey := range fullKeys {
		if len(fullKey) > prefixLen {
			businessKeys = append(businessKeys, TKey(fullKey[prefixLen:]))
		}
	}

	return businessKeys, nil
}

// Clear removes all keys in this namespace.
func (c *client[TKey]) Clear(ctx context.Context) error {
	err := c.driver.Clear(ctx, c.prefix)
	if err != nil {
		c.logf("error", ctx, "Clear failed: %v", err)
	}
	return err
}

// Incr atomically increments the integer value of a key by delta.
func (c *client[TKey]) Incr(ctx context.Context, key TKey, delta int64) (int64, error) {
	val, err := c.driver.Incr(ctx, c.key(key), delta)
	if err != nil {
		c.logf("error", ctx, "Incr %s failed: %v", key, err)
	}
	return val, err
}

// Decr atomically decrements the integer value of a key by delta.
func (c *client[TKey]) Decr(ctx context.Context, key TKey, delta int64) (int64, error) {
	val, err := c.driver.Decr(ctx, c.key(key), delta)
	if err != nil {
		c.logf("error", ctx, "Decr %s failed: %v", key, err)
	}
	return val, err
}

// GetSet atomically sets a key to a new value and returns the old value.
func (c *client[TKey]) GetSet(ctx context.Context, key TKey, newValue []byte) ([]byte, error) {
	oldVal, err := c.driver.GetSet(ctx, c.key(key), newValue)
	if err != nil && !errors.Is(err, ErrNotFound) {
		c.logf("error", ctx, "GetSet %s failed: %v", key, err)
	}
	return oldVal, err
}

// CompareAndSwap atomically compares and swaps the value if it matches oldValue.
func (c *client[TKey]) CompareAndSwap(ctx context.Context, key TKey, oldValue, newValue []byte, ttl time.Duration) (bool, error) {
	ok, err := c.driver.CompareAndSwap(ctx, c.key(key), oldValue, newValue, ttl)
	if err != nil {
		c.logf("error", ctx, "CompareAndSwap %s failed: %v", key, err)
	}
	return ok, err
}
