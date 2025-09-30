# Basic Usage

This guide walks you through common usage patterns for NameStore.

## Creating a Client

```go
import (
    "context"
    "github.com/khicago/namestore"
)

// Create a client with default in-memory storage
client := namestore.New[string]("myapp", "users")

// Use context for cancellation/timeout
ctx := context.Background()
```

## Storing and Retrieving Data

### Simple Set/Get

```go
// Store a value
err := client.Set(ctx, "alice", []byte("user data"), 0)
if err != nil {
    panic(err)
}

// Retrieve the value
data, err := client.Get(ctx, "alice")
if err != nil {
    panic(err)
}
fmt.Println(string(data)) // Output: user data
```

### With Expiration

```go
// Store with 5-minute TTL
err := client.Set(ctx, "session:abc", []byte("session data"), 5*time.Minute)

// Check remaining TTL
ttl, err := client.TTL(ctx, "session:abc")
fmt.Printf("Expires in: %v\n", ttl)

// Extend TTL
err = client.Expire(ctx, "session:abc", 10*time.Minute)

// Remove expiration
err = client.Persist(ctx, "session:abc")
```

## Conditional Operations

### SetNX - Set If Not Exists

```go
// Try to acquire a lock
acquired, err := client.SetNX(ctx, "lock:resource", []byte("owner1"), 30*time.Second)
if acquired {
    fmt.Println("Lock acquired!")
    defer client.Delete(ctx, "lock:resource")
    // ... do work
} else {
    fmt.Println("Lock already held")
}
```

## Batch Operations

### Multi-Get

```go
// Get multiple keys at once
results, err := client.MGet(ctx, "user:1", "user:2", "user:3")
if err != nil {
    panic(err)
}

for key, value := range results {
    fmt.Printf("%s: %s\n", key, value)
}
```

### Multi-Set

```go
// Set multiple keys with the same TTL
pairs := map[string][]byte{
    "user:1": []byte("alice"),
    "user:2": []byte("bob"),
    "user:3": []byte("charlie"),
}

err := client.MSet(ctx, pairs, 10*time.Minute)
```

### Multi-Delete

```go
// Delete multiple keys
err := client.MDel(ctx, "user:1", "user:2", "user:3")
```

## Atomic Operations

### Counter

```go
// Initialize counter
client.Set(ctx, "visitors", []byte{0,0,0,0,0,0,0,0}, 0)

// Increment
count, err := client.Incr(ctx, "visitors", 1)
fmt.Printf("Visitor count: %d\n", count)

// Decrement
count, err = client.Decr(ctx, "visitors", 1)
```

### GetSet - Atomic Swap

```go
// Set new value and get old value atomically
oldValue, err := client.GetSet(ctx, "config", []byte("new config"))
if err == namestore.ErrNotFound {
    fmt.Println("Key didn't exist before")
} else {
    fmt.Printf("Old value: %s\n", oldValue)
}
```

### Compare-And-Swap

```go
// Update only if current value matches
currentValue, _ := client.Get(ctx, "version")

success, err := client.CompareAndSwap(
    ctx,
    "version",
    currentValue,           // oldValue - must match
    []byte("new version"),  // newValue - to set
    0,                      // ttl
)

if success {
    fmt.Println("Version updated!")
} else {
    fmt.Println("Version changed by someone else")
}
```

## Namespace Operations

### List Keys

```go
// Get all keys in namespace
allKeys, err := client.Keys(ctx, "*")

// Get keys matching pattern
userKeys, err := client.Keys(ctx, "user:*")
adminKeys, err := client.Keys(ctx, "admin:*")
```

### Clear Namespace

```go
// Delete all keys in this namespace
err := client.Clear(ctx)
```

## Error Handling

```go
import "errors"

data, err := client.Get(ctx, "key")
if errors.Is(err, namestore.ErrNotFound) {
    // Key doesn't exist or has expired
    fmt.Println("Key not found")
} else if err != nil {
    // Other error
    return err
}

// Use data
fmt.Println(string(data))
```

## Type-Safe Keys

```go
// Define custom key type
type UserID string
type SessionID string

// Create clients with different key types
userClient := namestore.New[UserID]("app", "users")
sessionClient := namestore.New[SessionID]("app", "sessions")

// Type safety at compile time
userClient.Set(ctx, UserID("alice"), data, 0)      // ✓ OK
sessionClient.Set(ctx, SessionID("xyz"), data, 0)  // ✓ OK

// userClient.Set(ctx, SessionID("xyz"), data, 0)  // ✗ Compile error
```

## Working with JSON

```go
import "encoding/json"

type User struct {
    Name  string
    Email string
}

// Store JSON
user := User{Name: "Alice", Email: "alice@example.com"}
data, _ := json.Marshal(user)
client.Set(ctx, "user:alice", data, 0)

// Retrieve JSON
data, _ := client.Get(ctx, "user:alice")
var retrieved User
json.Unmarshal(data, &retrieved)
fmt.Printf("%+v\n", retrieved)
```

## Context Usage

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

data, err := client.Get(ctx, "key")
if errors.Is(err, context.DeadlineExceeded) {
    fmt.Println("Operation timed out")
}

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    time.Sleep(1 * time.Second)
    cancel()
}()

err := client.Set(ctx, "key", data, 0)
if errors.Is(err, context.Canceled) {
    fmt.Println("Operation canceled")
}
```

## Next Steps

- [Architecture Overview](architecture.md)
- [Client API Reference](api/client.md)
- [Custom Driver Implementation](advanced/custom-drivers.md)
- [Best Practices](advanced/best-practices.md)