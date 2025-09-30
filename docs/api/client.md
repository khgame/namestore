# Client API Reference

The `Client` interface provides all operations for interacting with namespaced key-value storage.

## Type Signature

```go
type Client[TKey ~string] interface {
    // Basic operations
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
```

## Creating a Client

### New

```go
func New[TKey ~string](rootNS, domain string, opts ...Option[TKey]) Client[TKey]
```

Creates a new namespaced client. Keys are stored with the prefix `rootNS:domain:`.

**Parameters:**
- `rootNS`: Root namespace (e.g., application name)
- `domain`: Sub-namespace (e.g., feature area)
- `opts`: Optional configuration (see [WithDriver](#withdriver))

**Example:**

```go
// Simple client with default in-memory driver
client := namestore.New[string]("myapp", "users")

// Client with custom key type
type UserID string
userClient := namestore.New[UserID]("myapp", "users")

// Client with custom driver
client := namestore.New[string]("myapp", "users",
    namestore.WithDriver[string](myRedisDriver))
```

### WithDriver

```go
func WithDriver[TKey ~string](d Driver) Option[TKey]
```

Specifies a custom storage driver. If not provided, the built-in in-memory driver is used.

**Example:**

```go
driver := NewRedisDriver("localhost:6379")
client := namestore.New[string]("app", "cache",
    namestore.WithDriver[string](driver))
```

## Basic Operations

### Set

```go
Set(ctx context.Context, key TKey, value []byte, ttl time.Duration) error
```

Sets a key to a value with optional TTL.

**Parameters:**
- `key`: Business key (will be prefixed with namespace)
- `value`: Data to store
- `ttl`: Time-to-live (0 for no expiration)

**Example:**

```go
// Set without expiration
err := client.Set(ctx, "user:123", []byte("alice"), 0)

// Set with 5-minute TTL
err := client.Set(ctx, "session:abc", []byte("data"), 5*time.Minute)
```

**Philosophy Note:** The `Set` method is **faithful (信)** to standard KV semantics - it always overwrites existing values. Use `SetNX` for conditional writes.

### SetNX

```go
SetNX(ctx context.Context, key TKey, value []byte, ttl time.Duration) (bool, error)
```

Sets a key only if it doesn't exist ("set if not exists").

**Returns:**
- `true` if the key was set
- `false` if the key already exists

**Example:**

```go
success, err := client.SetNX(ctx, "lock:resource", []byte("owner"), 30*time.Second)
if success {
    // Lock acquired
    defer client.Delete(ctx, "lock:resource")
    // ... critical section
}
```

**Use Case:** Distributed locking, preventing duplicate operations.

### Get

```go
Get(ctx context.Context, key TKey) ([]byte, error)
```

Retrieves the value for a key.

**Returns:**
- Value bytes
- `ErrNotFound` if key doesn't exist or has expired

**Example:**

```go
data, err := client.Get(ctx, "user:123")
if errors.Is(err, namestore.ErrNotFound) {
    // Key doesn't exist
}
```

### Delete

```go
Delete(ctx context.Context, key TKey) error
```

Deletes a key. Returns `nil` even if key doesn't exist (idempotent).

**Example:**

```go
err := client.Delete(ctx, "temp:data")
```

### Exists

```go
Exists(ctx context.Context, key TKey) (bool, error)
```

Checks if a key exists and is not expired.

**Example:**

```go
exists, err := client.Exists(ctx, "user:123")
if exists {
    fmt.Println("User exists")
}
```

## Error Handling

```go
var (
    ErrNotFound       = errors.New("namestore: not found")
    ErrTypeMismatch   = errors.New("namestore: type mismatch")
    ErrInvalidPattern = errors.New("namestore: invalid pattern")
)
```

**Best Practice:**

```go
data, err := client.Get(ctx, key)
if errors.Is(err, namestore.ErrNotFound) {
    // Handle missing key
    return nil
}
if err != nil {
    // Handle other errors
    return err
}
// Use data
```

## Next Steps

- [Batch Operations API](batch.md)
- [TTL Management API](ttl.md)
- [Atomic Operations API](atomic.md)