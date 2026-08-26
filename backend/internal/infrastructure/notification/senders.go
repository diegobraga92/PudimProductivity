// Package notification implements the Phase 3 notifications worker: it consumes
// task events from the event bus and delivers them via the configured senders
// (push via Firebase Cloud Messaging).
package notification

import (
	"context"
)

// PushDeliverer delivers a push notification to a device registration token.
type PushDeliverer interface {
	SendPush(ctx context.Context, token, title, body string) error
}

// NoopSender is a fallback used when no real sender is configured. It returns
// nil (success) so the pipeline stays healthy during local development.
type NoopSender struct{}

func (NoopSender) SendPush(context.Context, string, string, string) error { return nil }
