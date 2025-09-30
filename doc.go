// Package namestore provides a pluggable namespaced key-value storage toolkit with type-safe operations.
//
// # Overview
//
// namestore offers type-safe, namespace-isolated KV storage with pluggable backends.
// It separates concerns between business logic (Client) and storage implementation (Driver),
// following SOLID principles for maintainability and extensibility.
//
// # Architecture
//
// The package consists of two main abstractions:
//
// 1. Client[TKey]: Namespace-scoped interface for business operations
// 2. Driver: Storage backend interface for persistence
//
// Keys are automatically prefixed as "rootNS:domain:businessKey" to prevent collisions.
//
// # Quick Start
//
//	client := namestore.New[string]("myapp", "users")
//	ctx := context.Background()
//
//	// Basic operations
//	client.Set(ctx, "user:1001", []byte("Alice"), 1*time.Hour)
//	data, _ := client.Get(ctx, "user:1001")
//
//	// Atomic operations
//	views, _ := client.Incr(ctx, "page:views", 1)
//
// # Type-Safe Keys
//
// Use custom types for compile-time key validation:
//
//	type UserID string
//	type SessionID string
//
//	users := namestore.New[UserID]("myapp", "users")
//	sessions := namestore.New[SessionID]("myapp", "sessions")
//
//	users.Set(ctx, UserID("1001"), []byte("Alice"), 0)
//	// Compile error: sessions.Set(ctx, UserID("1001"), ...)
//
// # Custom Drivers
//
// Implement the Driver interface to support custom storage backends:
//
//	type myDriver struct { /* ... */ }
//
//	func (d *myDriver) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
//	    // Implementation
//	}
//	// Implement other Driver methods...
//
//	client := namestore.New[string]("myapp", "cache",
//	    namestore.WithDriver[string](&myDriver{}))
//
// # Thread Safety
//
// All operations are thread-safe. Multiple goroutines can safely share
// the same Client instance. Driver implementations must also be thread-safe.
//
// # Error Handling
//
// The package defines sentinel errors for common cases:
//
//	_, err := client.Get(ctx, "missing")
//	if errors.Is(err, namestore.ErrNotFound) {
//	    // Handle missing key
//	}
//
// Available errors: ErrNotFound, ErrTypeMismatch, ErrInvalidPattern
package namestore
