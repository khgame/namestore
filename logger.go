package namestore

import "context"

// Logger defines an interface for logging operations.
// Implementations should be safe for concurrent use.
type Logger interface {
	// Info logs informational messages
	Info(ctx context.Context, format string, args ...interface{})

	// Warn logs warning messages
	Warn(ctx context.Context, format string, args ...interface{})

	// Error logs error messages
	Error(ctx context.Context, format string, args ...interface{})

	// Debug logs debug messages
	Debug(ctx context.Context, format string, args ...interface{})
}

// noopLogger is a Logger that does nothing.
type noopLogger struct{}

func (noopLogger) Info(ctx context.Context, format string, args ...interface{})  {}
func (noopLogger) Warn(ctx context.Context, format string, args ...interface{})  {}
func (noopLogger) Error(ctx context.Context, format string, args ...interface{}) {}
func (noopLogger) Debug(ctx context.Context, format string, args ...interface{}) {}

var defaultLogger Logger = noopLogger{}
