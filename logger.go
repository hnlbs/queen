package queen

import "context"

// Logger is a structured logging interface compatible with slog.Logger.
type Logger interface {
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type noopLogger struct{}

func (n *noopLogger) InfoContext(ctx context.Context, msg string, args ...any)  {}
func (n *noopLogger) WarnContext(ctx context.Context, msg string, args ...any)  {}
func (n *noopLogger) ErrorContext(ctx context.Context, msg string, args ...any) {}

func defaultLogger() Logger {
	return &noopLogger{}
}
