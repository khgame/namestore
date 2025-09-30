# namestore

[![Go Reference](https://pkg.go.dev/badge/github.com/khicago/namestore.svg)](https://pkg.go.dev/github.com/khicago/namestore)
[![Go Report Card](https://goreportcard.com/badge/github.com/khicago/namestore)](https://goreportcard.com/report/github.com/khicago/namestore)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)](https://github.com/khicago/namestore)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/khicago/namestore)](https://github.com/khicago/namestore)
[![CI Status](https://github.com/khicago/namestore/workflows/CI/badge.svg)](https://github.com/khicago/namestore/actions)
[![Documentation](https://img.shields.io/badge/docs-github.io-blue)](https://khicago.github.io/namestore/)

Pluggable namespaced key-value storage toolkit for Go with type-safe keys and comprehensive operations.

## Features

- **Namespace Isolation**: Automatic key prefixing prevents collisions across domains
- **Type-Safe Keys**: Generic `Client[TKey]` with compile-time key type checking
- **Pluggable Drivers**: Abstract `Driver` interface for multiple storage backends
- **Rich Operations**:
  - Basic KV: Set, SetNX, Get, Delete, Exists
  - Batch: MGet, MSet, MDel
  - TTL Management: TTL, Expire, Persist
  - Atomic: Incr, Decr, GetSet, CompareAndSwap
  - Namespace: Keys (with pattern matching), Clear
- **Thread-Safe**: All operations are concurrency-safe
- **Zero Dependencies**: Pure Go with comprehensive test coverage (100%)

## Architecture

```
┌─────────────────────────────────────┐
│   Client[TKey] (Namespace Layer)   │
│  - Auto prefix: rootNS:domain:key  │
│  - Type-safe business keys          │
└────────────────┬────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────┐
│      Driver Interface (Storage)     │
│  - Memory (built-in)                │
│  - Redis (implement)                │
│  - Memcached (implement)            │
│  - Custom backends                  │
└─────────────────────────────────────┘
```

### Design Principles

1. **SRP (Single Responsibility)**: Client handles namespacing, Driver handles storage
2. **DIP (Dependency Inversion)**: Depend on `Driver` interface, not implementations
3. **OCP (Open/Closed)**: Extend via new Driver implementations without modifying core
4. **Type Safety**: Leverage Go generics for compile-time key validation

## Installation

```bash
go get github.com/khicago/namestore
```

**Requirements:**
- Go 1.18+ (for generics support)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/khicago/namestore"
)

func main() {
    // Create a namespaced client (uses in-memory driver by default)
    client := namestore.New[string]("myapp", "users")

    ctx := context.Background()

    // Set a value with TTL
    client.Set(ctx, "user:1001", []byte("Alice"), 1*time.Hour)

    // Get the value
    data, err := client.Get(ctx, "user:1001")
    if err != nil {
        panic(err)
    }
    fmt.Println(string(data)) // Output: Alice

    // Atomic counter
    views, _ := client.Incr(ctx, "page:views", 1)
    fmt.Println(views) // Output: 1
}
```

## Usage Examples

### Custom Key Types

```go
type UserID string
type SessionID string

// Type-safe clients for different domains
users := namestore.New[UserID]("myapp", "users")
sessions := namestore.New[SessionID]("myapp", "sessions")

users.Set(ctx, UserID("1001"), []byte("Alice"), 0)
sessions.Set(ctx, SessionID("abc123"), []byte("token"), 15*time.Minute)

// Compile error: type mismatch
// users.Set(ctx, SessionID("wrong"), []byte("data"), 0)
```

### Batch Operations

```go
// Batch set
pairs := map[string][]byte{
    "user:1001": []byte("Alice"),
    "user:1002": []byte("Bob"),
    "user:1003": []byte("Charlie"),
}
client.MSet(ctx, pairs, 1*time.Hour)

// Batch get
results, _ := client.MGet(ctx, "user:1001", "user:1002", "user:1003")
for key, value := range results {
    fmt.Printf("%s: %s\n", key, value)
}

// Batch delete
client.MDel(ctx, "user:1001", "user:1002")
```

### TTL Management

```go
// Set with TTL
client.Set(ctx, "session:abc", []byte("token"), 15*time.Minute)

// Check remaining TTL
ttl, _ := client.TTL(ctx, "session:abc")
fmt.Println(ttl) // e.g., 14m59s

// Extend TTL
client.Expire(ctx, "session:abc", 30*time.Minute)

// Make permanent
client.Persist(ctx, "session:abc")
```

### Atomic Operations

```go
// Counter
views, _ := client.Incr(ctx, "page:views", 1)
client.Decr(ctx, "page:views", 1)

// GetSet: atomic swap
oldToken, _ := client.GetSet(ctx, "user:token", []byte("new-token"))
fmt.Println(string(oldToken)) // previous token

// CompareAndSwap: optimistic locking
success, _ := client.CompareAndSwap(
    ctx,
    "config:version",
    []byte("v1.0"), // old value
    []byte("v1.1"), // new value
    0,              // no TTL
)
if success {
    fmt.Println("Version updated successfully")
}
```

### Namespace Operations

```go
// Store multiple keys
client.Set(ctx, "user:1001", []byte("Alice"), 0)
client.Set(ctx, "user:1002", []byte("Bob"), 0)
client.Set(ctx, "admin:root", []byte("Admin"), 0)

// List keys matching pattern
userKeys, _ := client.Keys(ctx, "user:*")
fmt.Println(userKeys) // [user:1001, user:1002]

// Clear entire namespace
client.Clear(ctx) // Removes all keys in this namespace
```

### Custom Driver Implementation

Implement the `Driver` interface to support other storage backends:

```go
type redisDriver struct {
    client *redis.Client
}

func (r *redisDriver) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *redisDriver) Get(ctx context.Context, key string) ([]byte, error) {
    data, err := r.client.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, namestore.ErrNotFound
    }
    return data, err
}

// Implement other methods...

// Use with namestore
driver := &redisDriver{client: redisClient}
client := namestore.New[string]("myapp", "cache", namestore.WithDriver[string](driver))
```

## Error Handling

```go
import "errors"

data, err := client.Get(ctx, "missing-key")
if errors.Is(err, namestore.ErrNotFound) {
    // Key doesn't exist
}

// Type mismatch (trying to Incr non-integer)
_, err = client.Incr(ctx, "string-key", 1)
if errors.Is(err, namestore.ErrTypeMismatch) {
    // Value is not an integer
}

// Invalid pattern
_, err = client.Keys(ctx, "[invalid")
if errors.Is(err, namestore.ErrInvalidPattern) {
    // Pattern syntax error
}
```

## Best Practices

### Namespace Design

```go
// Good: Clear hierarchy
users := namestore.New[string]("myapp", "users")
cache := namestore.New[string]("myapp", "cache")
sessions := namestore.New[string]("myapp", "sessions")

// Bad: No namespace isolation
global := namestore.New[string]("", "")
```

### Key Naming

```go
// Good: Descriptive, hierarchical keys
client.Set(ctx, "user:profile:1001", data, 0)
client.Set(ctx, "post:comments:5432", data, 0)

// Bad: Ambiguous keys
client.Set(ctx, "1001", data, 0)
client.Set(ctx, "data", data, 0)
```

### TTL Strategy

```go
// Good: Explicit TTL management
client.Set(ctx, "cache:expensive-query", result, 5*time.Minute)
client.Set(ctx, "session:token", token, 24*time.Hour)

// Bad: Everything permanent or everything expires
client.Set(ctx, "permanent-data", data, 1*time.Hour) // Should be 0
```

### Batch Operations

```go
// Good: Use batch operations for efficiency
results, _ := client.MGet(ctx, key1, key2, key3, key4, key5)

// Bad: Individual calls in loop
for _, key := range keys {
    client.Get(ctx, key) // Multiple round trips
}
```

## Testing

```bash
# Run tests
go test ./...

# With coverage
go test -cover ./...

# View coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Performance Characteristics

| Operation | Time Complexity | Notes |
|-----------|----------------|-------|
| Set/Get/Delete | O(1) | Constant time |
| SetNX | O(1) | Check + set |
| MGet/MSet/MDel | O(n) | n = number of keys |
| Incr/Decr | O(1) | Atomic operation |
| Keys | O(n) | n = total keys in namespace |
| Clear | O(n) | n = keys to delete |
| CompareAndSwap | O(1) | Atomic compare + swap |

**Memory Driver**: All operations hold a mutex lock, ensuring thread safety with minimal contention for read-heavy workloads.

## Thread Safety

All operations are thread-safe:
- **Client**: Stateless except for configuration (read-only after creation)
- **Driver**: Implementations must be thread-safe (memoryDriver uses `sync.RWMutex`)
- **Concurrent Access**: Multiple goroutines can safely share the same Client instance

## Documentation

Full documentation is available at [https://khicago.github.io/namestore/](https://khicago.github.io/namestore/)

- [Quick Start](https://khicago.github.io/namestore/#/README)
- [Architecture](https://khicago.github.io/namestore/#/architecture)
- [API Reference](https://khicago.github.io/namestore/#/api/client)
- [Design Principles](https://khicago.github.io/namestore/#/philosophy/design-principles)

## License

[MIT License](LICENSE)

## Contributing

Contributions welcome! Please ensure:
1. All tests pass: `go test ./...`
2. Coverage maintained at 100%: `go test -cover ./...`
3. Code follows Go conventions: `go fmt ./...` and `go vet ./...`
4. Add tests for new features
5. Update documentation

## Roadmap

- Redis driver implementation
- Memcached driver implementation
- Metrics and observability hooks
- Transaction support (multi-key operations)
- Pub/Sub for key change notifications
- LRU eviction policy for memory driver
- Background TTL cleanup for memory driver