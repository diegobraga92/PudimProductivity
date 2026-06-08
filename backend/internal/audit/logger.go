package audit

import "context"

// Logger is the interface used by domain services to record audit events.
// This allows injecting a no-op implementation for tests or when audit is disabled.
type Logger interface {
	// Log records an audit event asynchronously.
	Log(ctx context.Context, action, resource, resourceID string, oldValues, newValues any)
}

// NoopLogger is a logger that discards all events (used when audit is not configured).
type NoopLogger struct{}

func (NoopLogger) Log(_ context.Context, _ string, _ string, _ string, _ any, _ any) {
	// no-op
}