# Design Principles

NameStore is built on a foundation of software engineering principles and philosophical clarity.

## SOLID Principles

### Single Responsibility Principle (SRP)

Each component has one reason to change:

- **Client**: Manages namespace scoping and business key translation
- **Driver**: Handles actual storage operations
- **Entry**: Represents a single stored value with metadata

```go
// Client focuses on namespace management
type client[TKey ~string] struct {
    prefix string  // Responsibility: namespace
    driver Driver  // Responsibility: delegation
}

// Driver focuses on storage
type Driver interface {
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Get(ctx context.Context, key string) ([]byte, error)
    // ... storage operations only
}
```

### Open/Closed Principle (OCP)

**Open for extension, closed for modification:**

```go
// Extend by implementing Driver interface
type RedisDriver struct { /* ... */ }
func (r *RedisDriver) Set(ctx, key, value, ttl) error { /* ... */ }

// No need to modify Client code
client := namestore.New[string]("app", "cache",
    namestore.WithDriver[string](redisDriver))
```

### Liskov Substitution Principle (LSP)

Any `Driver` implementation can be used interchangeably:

```go
func testWithDriver(d namestore.Driver) {
    client := namestore.New[string]("test", "ns",
        namestore.WithDriver[string](d))

    // Works with Memory, Redis, or any other driver
    client.Set(ctx, "key", []byte("value"), 0)
}

testWithDriver(namestore.NewMemory())
testWithDriver(NewRedisDriver(...))
```

### Interface Segregation Principle (ISP)

The `Driver` interface is focused - clients only depend on what they need:

```go
// If you only need basic operations, you can define a minimal interface
type BasicDriver interface {
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Get(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
}

// Full Driver extends this with batch, TTL, atomic operations
```

### Dependency Inversion Principle (DIP)

High-level policy (Client) doesn't depend on low-level details (Memory, Redis):

```
       Client
         ↓ (depends on)
    Driver interface ← abstraction
         ↑ (implements)
    Memory / Redis ← concrete implementations
```

Both depend on the abstraction (`Driver`), not on each other.

## DRY (Don't Repeat Yourself)

### Key Prefixing Centralized

```go
// Single source of truth for namespace prefixing
func (c *client[TKey]) key(k TKey) string {
    return fmt.Sprintf("%s:%s", c.prefix, k)
}

// Used consistently everywhere
func (c *client[TKey]) Get(ctx context.Context, key TKey) ([]byte, error) {
    return c.driver.Get(ctx, c.key(key))
}
```

**Anti-pattern (avoided):**
```go
// ❌ Don't repeat prefixing logic
c.driver.Get(ctx, fmt.Sprintf("%s:%s", c.prefix, key))
```

### Value Cloning Utility

```go
// Single implementation for defensive copying
func clone(src []byte) []byte {
    if len(src) == 0 {
        return nil
    }
    dst := make([]byte, len(src))
    copy(dst, src)
    return dst
}

// Used in Get, Set, MGet, etc.
```

## Philosophical Clarity: 信达雅

### 信 (Faithfulness) - Semantic Correctness

**True to domain concepts:**

- `Set` always overwrites (faithful to KV store semantics)
- `SetNX` returns `bool` (faithful to "set if not exists" meaning)
- `TTL` returns `-1` for no expiration (faithful to Redis convention)
- `Incr`/`Decr` require 8-byte values (faithful to int64 representation)

**Example:**
```go
// Faithful: Clear means "remove all keys in this namespace"
func (c *client[TKey]) Clear(ctx context.Context) error {
    return c.driver.Clear(ctx, c.prefix)  // Only affects this namespace
}
```

### 达 (Expressiveness) - Clear Intent

**Method names convey purpose:**

- `MGet` clearly means "multi-get" (batch retrieval)
- `Persist` clearly means "remove expiration"
- `CompareAndSwap` clearly describes the atomic operation

**Parameter names are semantic:**
```go
// ✓ Expressive
func Expire(ctx context.Context, key TKey, ttl time.Duration) error

// ✗ Not expressive
func Expire(ctx context.Context, k TKey, d time.Duration) error
```

**Example:**
```go
// Expressive: The code reads like the requirement
ok, err := client.CompareAndSwap(ctx, "lock", oldOwner, newOwner, 30*time.Second)
// "Compare lock with oldOwner and swap to newOwner with 30s TTL"
```

### 雅 (Elegance) - Simplicity and Beauty

**Minimal API surface:**

- Generic `Client[TKey]` handles all key types
- Options pattern avoids constructor explosion
- Consistent error handling with sentinel errors

**Composable operations:**
```go
// Elegant: Build complex patterns from simple primitives
func acquireLock(client Client[string], resource string) bool {
    return client.SetNX(ctx, "lock:"+resource, owner, 30*time.Second)
}

func renewLock(client Client[string], resource string) error {
    return client.Expire(ctx, "lock:"+resource, 30*time.Second)
}
```

**Elegant error handling:**
```go
// Single check for "not found" across all operations
if errors.Is(err, namestore.ErrNotFound) {
    // Handle uniformly
}
```

## Concurrency Safety

### Thread-Safe by Contract

The `Driver` interface contract **requires** thread-safety:

```go
// From documentation:
// "Implementations must be thread-safe."
type Driver interface { /* ... */ }
```

### In-Memory Implementation

Uses `sync.RWMutex` for correct concurrent access:

```go
type Memory struct {
    mu   sync.RWMutex          // Protects data
    data map[string]entry      // Shared state
}

func (m *Memory) Get(ctx context.Context, key string) ([]byte, error) {
    m.mu.Lock()               // Exclusive lock for write
    defer m.mu.Unlock()
    // ... check expiration, delete if expired
}
```

**Why RWMutex?** Allows multiple concurrent readers, which is common in cache scenarios.

## Testability

### Interface-Based Design

Easy to mock for testing:

```go
type mockDriver struct {
    setFunc func(ctx, key, value, ttl) error
}

func (m *mockDriver) Set(ctx, key, value, ttl) error {
    return m.setFunc(ctx, key, value, ttl)
}

// Test with mock
client := namestore.New[string]("test", "ns",
    namestore.WithDriver[string](&mockDriver{
        setFunc: func(...) error { return nil },
    }))
```

### 100% Test Coverage

All code paths tested, including:
- Happy paths
- Error conditions
- Edge cases (expired keys, empty batches, concurrent access)
- Type mismatches for atomic operations

## Performance Considerations

### Batch Operations Reduce Round Trips

```go
// ✓ Efficient: Single call
results, _ := client.MGet(ctx, "key1", "key2", "key3")

// ✗ Inefficient: Multiple calls
data1, _ := client.Get(ctx, "key1")
data2, _ := client.Get(ctx, "key2")
data3, _ := client.Get(ctx, "key3")
```

### Lazy Expiration

Expired keys are deleted on access, not via background scanning:

```go
// Check expiration in Get
if entry.expired() {
    delete(m.data, key)  // Clean up on access
    return nil, ErrNotFound
}
```

**Trade-off:** Lower CPU overhead, but expired keys consume memory until accessed.

## Next Steps

- [Naming Convention Philosophy](naming.md)
- [Architecture Deep Dive](../architecture.md)
- [Best Practices](../advanced/best-practices.md)