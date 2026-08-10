// Package rabbitmq implements the eventbus.Bus interface over RabbitMQ, making
// it the durable backbone for asynchronous consumers (Phase 3). The real-time
// sync hub stays on the in-memory bus; a CompositeBus fans every event to both.
package rabbitmq

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
)

// Topology constants — exchange/queue names and dead-letter wiring.
const (
	ExchangeTaskEvents = "task.events"
	ExchangeDLX        = "task.events.dlx"
	QueueNotifications = "notifications"
	QueueDLQ           = "notifications.dlq"

	// RetryHeader is the AMQP header carrying how many times a message has been
	// retried via the DLQ pump.
	RetryHeader = "x-retry-count"
)

// Config tunes the adapter.
type Config struct {
	// URL is the AMQP connection URL (amqp://user:pass@host:port/).
	URL string
	// ReconnectDelay is the base backoff between connection attempts.
	ReconnectDelay time.Duration
	// Prefetch limits how many unacked messages a consumer holds at once.
	Prefetch int
	// MaxRetries is how many times a failing message is retried via the DLQ
	// before it is discarded. Default 5.
	MaxRetries int
}

// Bus is a RabbitMQ-backed eventbus.Bus. Publish sends the event to the
// task.events fanout exchange with W3C trace headers; Subscribe starts a
// consumer on the notifications queue and invokes the handler per message.
type Bus struct {
	cfg      Config
	mu       sync.Mutex
	conn     *amqp.Connection
	ch       *amqp.Channel // shared publishing channel
	done     chan struct{}
	closed   bool
	handlers map[uint64]func(context.Context, eventbus.Event) error
	nextSub  uint64
}

// New creates a Bus and declares the exchange/queue topology. It connects
// eagerly; if RabbitMQ is unavailable, the caller decides how to proceed.
func New(ctx context.Context, cfg Config) (*Bus, error) {
	if cfg.ReconnectDelay <= 0 {
		cfg.ReconnectDelay = 2 * time.Second
	}
	if cfg.Prefetch <= 0 {
		cfg.Prefetch = 10
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5
	}
	b := &Bus{
		cfg:      cfg,
		done:     make(chan struct{}),
		handlers: make(map[uint64]func(context.Context, eventbus.Event) error),
	}
	if err := b.connect(); err != nil {
		return nil, err
	}
	b.startRetryPump(ctx)
	return b, nil
}

// connect establishes the connection and channel and declares the topology.
func (b *Bus) connect() error {
	conn, err := amqp.Dial(b.cfg.URL)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := ch.Qos(b.cfg.Prefetch, 0, false); err != nil {
		_ = conn.Close()
		return err
	}
	if err := declareTopology(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}

	b.mu.Lock()
	b.conn, b.ch = conn, ch
	b.mu.Unlock()

	// Detect connection loss so consumer loops can be torn down and recreated.
	go func() {
		_, ok := <-conn.NotifyClose(make(chan *amqp.Error, 1))
		if !ok {
			return
		}
		b.mu.Lock()
		closed := b.closed
		b.mu.Unlock()
		if !closed {
			log.Warn().Msg("rabbitmq connection lost")
		}
	}()

	return nil
}

// declareTopology creates the fanout exchange and the notifications queue
// (which dead-letters failed messages). Retries are handled by the adapter's
// DLQ retry pump — not by queue-level TTL loops, which RabbitMQ's dead-letter
// cycle protection can drop.
func declareTopology(ch *amqp.Channel) error {
	mainArgs := amqp.Table{
		"x-dead-letter-exchange": ExchangeDLX,
	}
	if err := ch.ExchangeDeclare(ExchangeTaskEvents, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(ExchangeDLX, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(QueueNotifications, true, false, false, false, mainArgs); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(QueueDLQ, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(QueueNotifications, "", ExchangeTaskEvents, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(QueueDLQ, "", ExchangeDLX, false, nil); err != nil {
		return err
	}
	return nil
}

// Publish marshals the event and sends it to the exchange with trace headers.
// Each message gets a unique ID (the idempotency key consumers use to dedupe
// at-least-once redeliveries).
func (b *Bus) Publish(ctx context.Context, typ eventbus.EventType, payload interface{}) error {
	b.mu.Lock()
	ch := b.ch
	closed := b.closed
	b.mu.Unlock()
	if closed || ch == nil {
		return eventbus.ErrBusClosed
	}

	event := eventbus.Event{
		ID:        uuid.NewString(),
		Type:      typ,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	headers := amqp.Table{}
	injectTrace(ctx, headers)

	return ch.PublishWithContext(ctx, ExchangeTaskEvents, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		MessageId:    event.ID,
		DeliveryMode: amqp.Persistent,
		Timestamp:    event.Timestamp,
		Headers:      headers,
	})
}

// Subscribe registers a handler and starts a consumer goroutine on the
// notifications queue. The returned function cancels the subscription.
func (b *Bus) Subscribe(ctx context.Context, handler eventbus.Handler) (func(), error) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return func() {}, eventbus.ErrBusClosed
	}
	b.nextSub++
	subID := b.nextSub
	b.handlers[subID] = handler
	b.mu.Unlock()

	deliveries, err := b.consume(QueueNotifications)
	if err != nil {
		return func() {}, err
	}

	go b.dispatchLoop(ctx, subID, deliveries)

	return func() {
		b.mu.Lock()
		delete(b.handlers, subID)
		b.mu.Unlock()
	}, nil
}

// consume opens a consumer on the queue, returning the delivery channel.
func (b *Bus) consume(queue string) (<-chan amqp.Delivery, error) {
	b.mu.Lock()
	conn := b.conn
	b.mu.Unlock()
	if conn == nil {
		return nil, eventbus.ErrBusClosed
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Qos(b.cfg.Prefetch, 0, false); err != nil {
		return nil, err
	}
	return ch.Consume(queue, "", false, false, false, false, nil)
}

// dispatchLoop reads deliveries and hands them to the handler, acking on
// success and rejecting (→ dead-letter queue) on failure.
func (b *Bus) dispatchLoop(ctx context.Context, subID uint64, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.done:
			return
		case msg, ok := <-deliveries:
			if !ok {
				return // channel closed; a reconnect re-subscribes
			}
			b.handleDelivery(ctx, subID, msg)
		}
	}
}

func (b *Bus) handleDelivery(ctx context.Context, subID uint64, msg amqp.Delivery) {
	// Bounded retry: once a message has been dead-lettered (retried) more than
	// MaxRetries times, discard it instead of looping forever.
	if retryCount(msg.Headers) >= b.cfg.MaxRetries {
		log.Error().Str("message_id", msg.MessageId).
			Int("retries", retryCount(msg.Headers)).
			Msg("rabbitmq: message exceeded max retries, discarding")
		_ = msg.Ack(false)
		return
	}

	b.mu.Lock()
	handler, ok := b.handlers[subID]
	b.mu.Unlock()
	if !ok {
		_ = msg.Reject(false) // no subscriber anymore → DLQ
		return
	}

	var event eventbus.Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Error().Err(err).Msg("rabbitmq: failed to unmarshal event, rejecting")
		_ = msg.Reject(false)
		return
	}

	// Continue the producer's trace from the AMQP headers.
	hctx := extractTrace(ctx, msg.Headers)

	if err := handler(hctx, event); err != nil {
		log.Warn().Err(err).Str("event_type", string(event.Type)).Msg("rabbitmq: handler failed, sending to DLQ for retry")
		_ = msg.Reject(false)
		return
	}
	_ = msg.Ack(false)
}

// retryCount reads how many times a message has been retried. It prefers the
// explicit x-retry-count header set by the retry pump and falls back to the
// x-death count RabbitMQ sets when dead-lettering.
func retryCount(headers amqp.Table) int {
	if headers != nil {
		if raw, ok := headers[RetryHeader]; ok {
			switch n := raw.(type) {
			case int32:
				return int(n)
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
		}
	}

	raw, ok := headers["x-death"]
	if !ok {
		return 0
	}
	deaths, ok := raw.([]interface{})
	if !ok {
		return 0
	}
	for _, d := range deaths {
		table, ok := d.(amqp.Table)
		if !ok {
			continue
		}
		if c, ok := table["count"]; ok {
			switch n := c.(type) {
			case int32:
				return int(n)
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
		}
	}
	return 0
}

// startRetryPump consumes from the dead-letter queue and republishes messages
// back to the main exchange (bounded by MaxRetries). This is the retry loop;
// it avoids queue-level TTL loops that RabbitMQ's dead-letter cycle protection
// can silently drop.
func (b *Bus) startRetryPump(ctx context.Context) {
	deliveries, err := b.consume(QueueDLQ)
	if err != nil {
		log.Warn().Err(err).Msg("rabbitmq: failed to start DLQ retry pump")
		return
	}
	go func() {
		log.Info().Msg("rabbitmq: DLQ retry pump started")
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.done:
				return
			case msg, ok := <-deliveries:
				if !ok {
					return
				}
				b.retryFromDLQ(ctx, msg)
			}
		}
	}()
}

func (b *Bus) retryFromDLQ(ctx context.Context, msg amqp.Delivery) {
	count := retryCount(msg.Headers)
	if count >= b.cfg.MaxRetries {
		log.Error().Str("message_id", msg.MessageId).
			Int("retries", count).
			Msg("rabbitmq: message exceeded max retries, discarding")
		_ = msg.Ack(false)
		return
	}

	// Copy the original headers (including traceparent) and bump the counter.
	headers := amqp.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}
	headers[RetryHeader] = int32(count + 1)

	b.mu.Lock()
	ch := b.ch
	b.mu.Unlock()
	if ch == nil {
		_ = msg.Nack(false, true) // bus closed; requeue in DLQ
		return
	}

	err := ch.PublishWithContext(ctx, ExchangeTaskEvents, "", false, false, amqp.Publishing{
		ContentType:  msg.ContentType,
		Body:         msg.Body,
		MessageId:    msg.MessageId,
		DeliveryMode: amqp.Persistent,
		Timestamp:    msg.Timestamp,
		Headers:      headers,
	})
	if err != nil {
		log.Warn().Err(err).Str("message_id", msg.MessageId).Msg("rabbitmq: DLQ republish failed")
		_ = msg.Nack(false, true) // requeue in DLQ so the pump retries later
		return
	}
	_ = msg.Ack(false)
	log.Info().Str("message_id", msg.MessageId).
		Int("retry", count+1).
		Int("max", b.cfg.MaxRetries).
		Msg("rabbitmq: republished from DLQ for retry")
}

// Close tears down the connection and marks the bus closed.
func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	conn := b.conn
	b.conn = nil
	b.ch = nil
	close(b.done)
	b.mu.Unlock()

	if conn != nil {
		return conn.Close()
	}
	return nil
}
