# NameStore

> Pluggable namespaced key-value storage toolkit with comprehensive operations

## What is NameStore?

NameStore is a Go library that provides a **namespaced key-value storage abstraction** with automatic key prefixing, pluggable storage backends, and comprehensive operations including batch processing, TTL management, and atomic operations.

### Philosophy: 信达雅 (Faithfulness, Expressiveness, Elegance)

The design of NameStore follows three principles:

- **信 (Faithfulness)**: True to the domain - namespacing is transparent and predictable
- **达 (Expressiveness)**: Clear semantics - every operation conveys intent precisely
- **雅 (Elegance)**: Simple beauty - minimal API surface with maximum functionality

## Features

### 🎯 Namespaced by Default

Keys are automatically prefixed with `root:domain:` pattern, providing natural isolation:

```go
client := namestore.New[string]("myapp", "users")
client.Set(ctx, "alice", []byte("data"), 0)
// Actually stored as: "myapp:users:alice"
```

### 🔌 Pluggable Architecture

Interface-based design allows custom storage backends:

```go
type Driver interface {
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Get(ctx context.Context, key string) ([]byte, error)
    // ... more operations
}

client := namestore.New[string]("root", "domain",
    namestore.WithDriver[string](myCustomDriver))
```

### ⚡ Comprehensive Operations

**Basic Operations**
- `Set`, `SetNX`, `Get`, `Delete`, `Exists`

**Batch Operations**
- `MGet`, `MSet`, `MDel` - process multiple keys efficiently

**TTL Management**
- `TTL`, `Expire`, `Persist` - fine-grained expiration control

**Namespace Operations**
- `Keys`, `Clear` - pattern matching and bulk deletion

**Atomic Operations**
- `Incr`, `Decr` - atomic counters
- `GetSet` - atomic read-modify-write
- `CompareAndSwap` - lock-free concurrency

### 🔒 Thread-Safe

Built-in in-memory driver uses `sync.RWMutex` for safe concurrent access:

```go
// Safe to use from multiple goroutines
for i := 0; i < 100; i++ {
    go func() {
        client.Set(ctx, "key", data, 0)
    }()
}
```

### 🎨 Type-Safe Keys

Go generics provide compile-time type safety for business keys:

```go
type UserID string

client := namestore.New[UserID]("app", "users")
client.Set(ctx, UserID("alice"), data, 0)

// Type mismatch caught at compile time
// client.Set(ctx, 123, data, 0) // ❌ Compile error
```

## Quick Start

### Installation

```bash
go get github.com/khicago/namestore
```

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/khicago/namestore"
)

func main() {
    // Create a namespaced client
    client := namestore.New[string]("myapp", "cache")
    ctx := context.Background()

    // Set a value with TTL
    client.Set(ctx, "user:123", []byte("alice"), 5*time.Minute)

    // Get the value
    data, err := client.Get(ctx, "user:123")
    if err != nil {
        panic(err)
    }
    fmt.Println(string(data)) // Output: alice

    // Batch operations
    client.MSet(ctx, map[string][]byte{
        "user:124": []byte("bob"),
        "user:125": []byte("charlie"),
    }, 5*time.Minute)

    results, _ := client.MGet(ctx, "user:123", "user:124", "user:125")
    fmt.Printf("Got %d users\n", len(results)) // Output: Got 3 users

    // Atomic counter
    count, _ := client.Incr(ctx, "visitor_count", 1)
    fmt.Printf("Visitor count: %d\n", count)
}
```

## Architecture

```
┌─────────────────────────────────────────┐
│         Client[TKey ~string]            │
│  (Namespace: "root:domain")             │
│                                         │
│  • Set, Get, Delete, Exists             │
│  • MGet, MSet, MDel (batch)             │
│  • TTL, Expire, Persist                 │
│  • Keys, Clear (namespace)              │
│  • Incr, Decr, GetSet, CAS (atomic)     │
└─────────────────────────────────────────┘
              ↓ (uses)
┌─────────────────────────────────────────┐
│           Driver Interface              │
│  (Storage abstraction)                  │
│                                         │
│  • Pluggable backend                    │
│  • Thread-safe contract                 │
│  • Key prefixing transparent            │
└─────────────────────────────────────────┘
      ↓                    ↓
┌──────────────┐   ┌─────────────────┐
│   Memory     │   │  Custom Driver  │
│  (built-in)  │   │  (Redis, etc.)  │
└──────────────┘   └─────────────────┘
```

## Design Principles

### 1. Separation of Concerns (SRP)

- **Client**: Business logic + namespace management
- **Driver**: Storage implementation
- Each component has a single, well-defined responsibility

### 2. Dependency Inversion (DIP)

- Client depends on `Driver` interface, not concrete implementations
- Easy to swap storage backends without changing client code

### 3. Open/Closed Principle (OCP)

- Open for extension: implement custom drivers
- Closed for modification: core client logic is stable

### 4. Don't Repeat Yourself (DRY)

- Key prefixing logic centralized in `client.key()` method
- No code duplication across operations

### 5. Philosophical Clarity (信达雅)

- **信 (Faithful)**: Namespace semantics are predictable
- **达 (Expressive)**: Method names convey intent clearly
- **雅 (Elegant)**: Minimal, composable API

## Next Steps

- [Installation Guide](installation.md)
- [Basic Usage Tutorial](basic-usage.md)
- [Architecture Deep Dive](architecture.md)
- [API Reference](api/client.md)
- [Custom Driver Guide](advanced/custom-drivers.md)

## License

[Add your license here]

## Contributing

Contributions are welcome! Please read our contributing guidelines first.