// Package notification implements the Phase 3 notifications worker: it consumes
// task events from the event bus and delivers them via the configured senders
// (email via SMTP/Mailpit, push via Firebase Cloud Messaging).
package notification

import (
	"context"
)

// EmailDeliverer delivers an email. Implementations must be safe for concurrent
// use and should return an error only when the recipient did not receive it.
type EmailDeliverer interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// PushDeliverer delivers a push notification to a device registration token.
type PushDeliverer interface {
	SendPush(ctx context.Context, token, title, body string) error
}

// NoopSender is a fallback used when no real sender is configured. It returns
// nil (success) so the pipeline stays healthy during local development.
type NoopSender struct{}

func (NoopSender) SendEmail(context.Context, string, string, string) error { return nil }
func (NoopSender) SendPush(context.Context, string, string, string) error  { return nil }
