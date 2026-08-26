package notification

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

// Recipients configures who receives notifications. The app is single-user in
// this phase; per-user targeting arrives with auth (threat model P0 / Phase 8).
type Recipients struct {
	// PushToken is a device registration token for push notifications.
	PushToken string
}

// Worker consumes task events from the bus and delivers notifications through
// the configured senders. A failed send returns an error so the RabbitMQ
// adapter rejects the message into the dead-letter queue for retry.
type Worker struct {
	bus    eventbus.Bus
	pushes []PushDeliverer
	repo   Repo
	recip  Recipients
	unsub  func()
}

// NewWorker creates a notifications worker.
func NewWorker(bus eventbus.Bus, pushes []PushDeliverer, repo Repo, recip Recipients) *Worker {
	return &Worker{
		bus:    bus,
		pushes: pushes,
		repo:   repo,
		recip:  recip,
	}
}

// Run subscribes the worker to the bus and blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	unsub, err := w.bus.Subscribe(ctx, w.handleEvent)
	if err != nil {
		return fmt.Errorf("notification worker: subscribe: %w", err)
	}
	w.unsub = unsub
	log.Info().Str("push_token", w.recip.PushToken).Msg("notification worker started")
	<-ctx.Done()
	return nil
}

// Close unsubscribes the worker from the bus.
func (w *Worker) Close() {
	if w.unsub != nil {
		w.unsub()
	}
}

// handleEvent renders a notification for the event and sends it on every
// channel exactly once (idempotent via the Repo).
func (w *Worker) handleEvent(ctx context.Context, event eventbus.Event) error {
	title, body, ok := renderNotification(event)
	if !ok {
		return nil // not a notifiable event type
	}

	if w.recip.PushToken != "" {
		return w.sendOnce(ctx, event, "push", func() error {
			return w.sendPush(ctx, title, body)
		})
	}
	return nil
}

// sendOnce checks idempotency, sends via the given closure, and records the
// send. If the closure fails, nothing is recorded and the error propagates so
// the message can be retried via the DLQ.
func (w *Worker) sendOnce(ctx context.Context, event eventbus.Event, channel string, send func() error) error {
	eventID := event.ID
	if eventID == "" {
		eventID = fmt.Sprintf("%s/%d", event.Type, event.Seq)
	}

	already, err := w.repo.AlreadySent(ctx, eventID, channel)
	if err != nil {
		return fmt.Errorf("notification: dedup check failed: %w", err)
	}
	if already {
		return nil
	}

	if err := send(); err != nil {
		return err
	}

	log.Info().Ctx(ctx).Str("channel", channel).Str("event_id", eventID).Msg("notification sent")

	taskID := extractTaskID(event)
	return w.repo.MarkSent(ctx, eventID, channel, string(event.Type), taskID)
}

func (w *Worker) sendPush(ctx context.Context, title, body string) error {
	for _, s := range w.pushes {
		if err := s.SendPush(ctx, w.recip.PushToken, title, body); err != nil {
			log.Warn().Ctx(ctx).Err(err).Msg("push send failed")
			return err
		}
	}
	return nil
}
