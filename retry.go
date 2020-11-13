package retry

import "context"

// Logger is a simplified interface for logging, mainly used to decouple retry from logging.
type Logger interface {
	Error(ctx context.Context, msg string, keysAndValues ...interface{})
	Warn(ctx context.Context, msg string, keysAndValues ...interface{})
	Debug(ctx context.Context, msg string, keysAndValues ...interface{})
}
