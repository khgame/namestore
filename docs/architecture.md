# Architecture

This document explains the architecture and design of NameStore.

## Overview

NameStore follows a layered architecture with clear separation between business logic and storage implementation.

```
┌───────────────────────────────────────────────────┐
│                  Application                      │
│           (Your business logic)                   │
└───────────────────────────────────────────────────┘
                      ↓ uses
┌───────────────────────────────────────────────────┐
│              Client[TKey ~string]                 │
│                                                   │
│  • Namespace Management (root:domain:key)         │
│  • Business Key Translation                       │
│  • High-Level Operations                          │
│  • Type Safety via Generics                       │
└───────────────────────────────────────────────────┘
                      ↓ delegates to
┌───────────────────────────────────────────────────┐
│              Driver Interface                     │
│                                                   │
│  • Storage Abstraction                            │
│  • Thread-Safety Contract                         │
│  • Implementation-Agnostic                        │
└───────────────────────────────────────────────────┘
           ↓                    ↓                ↓
┌──────────────────┐  ┌──────────────────┐  ┌─────────────┐
│  Memory Driver   │  │  Redis Driver    │  │  Custom     │
│  (built-in)      │  │  (external)      │  │  Driver     │
└──────────────────┘  └──────────────────┘  └─────────────┘
```

## Components

### 1. Client Layer

**Responsibility:** Business logic and namespace management

```go
type client[TKey ~string] struct {
    prefix string    // Namespace prefix: "root:domain"
    driver Driver    // Storage backend
}
```

**Key Functions:**
- Translate business keys to storage keys
- Manage namespace isolation
- Provide type-safe API via generics
- Delegate storage operations to driver

**Example:**
```go
client := namestore.New[string]("myapp", "users")
// client.prefix = "myapp:users"

client.Set(ctx, "alice", data, 0)
// Translates to: driver.Set(ctx, "myapp:users:alice", data, 0)
```

### 2. Driver Interface

**Responsibility:** Storage abstraction

```go
type Driver interface {
    // Basic operations
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Get(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
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
```

**Contract:**
- **Thread-safe**: Must handle concurrent access correctly
- **Context-aware**: Respect cancellation and timeouts
- **Error semantics**: Return `ErrNotFound` for missing keys

### 3. Memory Driver (Built-in)

**Responsibility:** In-memory storage implementation

```go
type Memory struct {
    mu   sync.RWMutex
    data map[string]entry
}

type entry struct {
    value  []byte
    expire time.Time
}
```

**Features:**
- Thread-safe via `sync.RWMutex`
- Lazy expiration (clean on access)
- Defensive copying (values are cloned)
- Zero dependencies

**Performance:**
- O(1) for Get, Set, Delete, Exists
- O(n) for Keys, Clear (where n = total keys)
- O(k) for batch operations (where k = batch size)

## Data Flow

### Set Operation

```
Application
    ↓ client.Set(ctx, "alice", data, 0)
Client
    ↓ key("alice") → "myapp:users:alice"
    ↓ driver.Set(ctx, "myapp:users:alice", data, 0)
Memory Driver
    ↓ mu.Lock()
    ↓ data["myapp:users:alice"] = entry{clone(data), expiry(0)}
    ↓ mu.Unlock()
    ↓ return nil
```

### Get Operation

```
Application
    ↓ client.Get(ctx, "alice")
Client
    ↓ key("alice") → "myapp:users:alice"
    ↓ driver.Get(ctx, "myapp:users:alice")
Memory Driver
    ↓ mu.Lock()
    ↓ e := data["myapp:users:alice"]
    ↓ if e.expired() { delete(data, key); return ErrNotFound }
    ↓ mu.Unlock()
    ↓ return clone(e.value), nil
```

### MGet Operation (Batch)

```
Application
    ↓ client.MGet(ctx, "alice", "bob", "charlie")
Client
    ↓ keys → ["myapp:users:alice", "myapp:users:bob", "myapp:users:charlie"]
    ↓ driver.MGet(ctx, keys)
Memory Driver
    ↓ mu.Lock()
    ↓ result := {}
    ↓ for each key: if exists && !expired { result[key] = clone(value) }
    ↓ mu.Unlock()
    ↓ return result
Client
    ↓ strip prefix → {"alice": data1, "bob": data2, "charlie": data3}
    ↓ return to application
```

## Namespace Isolation

NameStore uses **hierarchical key prefixing** for namespace isolation:

```
root:domain:businessKey
 ↑     ↑         ↑
 |     |         └─ Application-level key
 |     └─────────── Sub-namespace (feature/module)
 └─────────────────── Root namespace (application)
```

**Example:**

```go
// Different namespaces are isolated
userCache := namestore.New[string]("myapp", "users")
sessionCache := namestore.New[string]("myapp", "sessions")

userCache.Set(ctx, "alice", []byte("user data"), 0)
sessionCache.Set(ctx, "alice", []byte("session data"), 0)

// Stored as:
// "myapp:users:alice"    → "user data"
// "myapp:sessions:alice" → "session data"
```

**Benefits:**
1. **Logical separation**: Different domains don't collide
2. **Selective clearing**: `Clear()` only affects one namespace
3. **Pattern matching**: `Keys("user:*")` scoped to namespace
4. **Multi-tenancy**: Each tenant gets own root namespace

## Concurrency Model

### Memory Driver Locking Strategy

Uses **RWMutex** for read/write optimization:

```go
type Memory struct {
    mu   sync.RWMutex  // Read-write mutex
    data map[string]entry
}

// Read operations use RLock (multiple concurrent readers)
func (m *Memory) Get(ctx context.Context, key string) ([]byte, error) {
    m.mu.Lock()        // Still uses Lock() because we may delete expired keys
    defer m.mu.Unlock()
    // ...
}

// Write operations use Lock (exclusive access)
func (m *Memory) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    // ...
}
```

**Note:** Even read operations like `Get` use exclusive `Lock()` (not `RLock()`) because they may delete expired keys, which is a write operation.

### Defensive Copying

Values are **cloned** on read and write to prevent external mutation:

```go
// Set: Clone input to protect against caller mutation
m.data[key] = entry{value: clone(value), expire: expiry(ttl)}

// Get: Clone output to protect against caller mutation
return clone(entry.value), nil

func clone(src []byte) []byte {
    if len(src) == 0 {
        return nil
    }
    dst := make([]byte, len(src))
    copy(dst, src)
    return dst
}
```

## Expiration Strategy

**Lazy expiration** - keys are deleted when accessed:

```go
func (m *Memory) Get(ctx context.Context, key string) ([]byte, error) {
    entry, ok := m.data[key]
    if !ok || entry.expired() {
        delete(m.data, key)  // Clean up on access
        return nil, ErrNotFound
    }
    return clone(entry.value), nil
}

func (e entry) expired() bool {
    if e.expire.IsZero() {
        return false  // No expiration
    }
    return time.Now().After(e.expire)
}
```

**Trade-offs:**
- ✓ No background goroutines (simpler, less CPU)
- ✓ Expired keys cleaned up eventually
- ✗ Memory not freed until next access

**Alternative (not implemented):** Active expiration with background cleanup goroutine.

## Extension Points

### Custom Driver Implementation

Implement the `Driver` interface for custom backends:

```go
type RedisDriver struct {
    client *redis.Client
}

func (r *RedisDriver) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisDriver) Get(ctx context.Context, key string) ([]byte, error) {
    val, err := r.client.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, namestore.ErrNotFound
    }
    return val, err
}

// Implement remaining methods...
```

Usage:

```go
redisDriver := &RedisDriver{client: redisClient}
client := namestore.New[string]("app", "cache",
    namestore.WithDriver[string](redisDriver))
```

### Custom Key Types

Use Go generics for domain-specific key types:

```go
type UserID string
type SessionID string

userClient := namestore.New[UserID]("app", "users")
sessionClient := namestore.New[SessionID]("app", "sessions")

// Type safety enforced at compile time
userClient.Set(ctx, UserID("alice"), data, 0)  // ✓
sessionClient.Set(ctx, UserID("alice"), data, 0)  // ✗ Compile error
```

## Design Decisions

### Why Interface-Based?

**Decision:** Use `Driver` interface rather than concrete implementations.

**Rationale:**
- ✓ Testability (easy to mock)
- ✓ Extensibility (plug in custom storage)
- ✓ Separation of concerns (client doesn't know storage details)

### Why Generics for Keys?

**Decision:** `Client[TKey ~string]` instead of `Client` with `string` keys.

**Rationale:**
- ✓ Type safety (prevent mixing different key types)
- ✓ Self-documenting (intent clear from types)
- ✓ Zero runtime cost (compile-time feature)

### Why Lazy Expiration?

**Decision:** Delete expired keys on access, not via background job.

**Rationale:**
- ✓ Simpler implementation (no goroutines)
- ✓ Lower CPU overhead (no periodic scanning)
- ✗ Memory not freed immediately (acceptable trade-off)

### Why Defensive Copying?

**Decision:** Clone byte slices on read/write.

**Rationale:**
- ✓ Prevents accidental mutation by callers
- ✓ Thread safety (no shared mutable state)
- ✗ Slight performance cost (acceptable for correctness)

## Next Steps

- [Namespace Design Details](namespace-design.md)
- [Driver Interface Specification](driver-interface.md)
- [Custom Driver Implementation Guide](advanced/custom-drivers.md)