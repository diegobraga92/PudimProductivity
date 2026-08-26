package eventbus

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/pkg/uuid"
)

// DefaultRedisChannel is the pub/sub channel used to fan events out to every
// backend instance.
const DefaultRedisChannel = "pudim:events"

// RedisConfig tunes the Redis-backed bus.
type RedisConfig struct {
	// URL is a redis:// URL, e.g. "redis://localhost:6379/0".
	URL string
	// Channel is the pub/sub channel events are broadcast on. Empty means
	// DefaultRedisChannel.
	Channel string
	// InstanceID uniquely identifies this process. It is stamped on every
	// published message so subscribers can ignore their own echo. Empty means
	// a random UUID is generated.
	InstanceID string
}

// RedisBus implements Bus over Redis pub/sub so events produced on one backend
// instance are received by every other instance's sync hub.
// Messages published while a subscriber is disconnected are dropped
// (clients recover via the REST catch-up path).
type RedisBus struct {
	client  *redis.Client
	channel string
	origin  string

	mu     sync.RWMutex
	subs   []*redisSubscription
	closed bool
	cancel context.CancelFunc
	pubsub *redis.PubSub // nil until the first subscriber starts
	done   chan struct{}
}

type redisSubscription struct {
	handler Handler
	active  bool
}

// wireMessage is the on-the-wire envelope. Origin lets subscribers skip their
// own publishes (Redis pub/sub echoes messages back to the publisher).
type wireMessage struct {
	Origin string `json:"origin"`
	Event  Event  `json:"event"`
}

// NewRedisBus connects eagerly and verifies reachability with a ping. It
// returns an error when Redis is unavailable so the caller can degrade
// gracefully (see cmd/server wiring).
func NewRedisBus(ctx context.Context, cfg RedisConfig) (*RedisBus, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	if cfg.Channel == "" {
		cfg.Channel = DefaultRedisChannel
	}
	if cfg.InstanceID == "" {
		cfg.InstanceID = uuid.NewUUID()
	}

	return &RedisBus{
		client:  client,
		channel: cfg.Channel,
		origin:  cfg.InstanceID,
		done:    make(chan struct{}),
	}, nil
}

// Publish marshals the event and broadcasts it on the channel.
func (b *RedisBus) Publish(ctx context.Context, typ EventType, payload interface{}) error {
	msg := wireMessage{
		Origin: b.origin,
		Event: Event{
			Type:      typ,
			Timestamp: time.Now().UTC(),
			Payload:   payload,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	b.mu.RLock()
	closed := b.closed
	client := b.client
	b.mu.RUnlock()
	if closed || client == nil {
		return ErrBusClosed
	}
	return client.Publish(ctx, b.channel, data).Err()
}

// Subscribe registers a handler for remote events.
func (b *RedisBus) Subscribe(ctx context.Context, handler Handler) (func(), error) {
	if handler == nil {
		return func() {}, nil
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return func() {}, ErrBusClosed
	}
	sub := &redisSubscription{handler: handler, active: true}
	b.subs = append(b.subs, sub)
	first := len(b.subs) == 1
	b.mu.Unlock()

	if first {
		subCtx, cancel := context.WithCancel(ctx)
		pubsub := b.client.Subscribe(subCtx, b.channel)
		b.mu.Lock()
		b.cancel = cancel
		b.pubsub = pubsub
		b.mu.Unlock()
		go b.runSubscriber(subCtx, pubsub)
	}

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		sub.active = false
	}, nil
}

// runSubscriber consumes messages from the channel until the pubsub is closed.
func (b *RedisBus) runSubscriber(ctx context.Context, pubsub *redis.PubSub) {
	defer close(b.done)
	defer func() { _ = pubsub.Close() }()

	log.Info().Str("channel", b.channel).Str("instance_id", b.origin).Msg("redis event subscriber started")
	for msg := range pubsub.Channel() {
		b.handleMessage(msg.Payload)
	}
	log.Info().Str("channel", b.channel).Msg("redis event subscriber stopped")
}

// handleMessage decodes a channel message, filters self-origin messages and
// dispatches the rest to the registered handlers.
func (b *RedisBus) handleMessage(raw string) {
	var msg wireMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		log.Warn().Err(err).Msg("redis event bus: dropped malformed message")
		return
	}
	if msg.Origin == b.origin {
		return
	}

	b.mu.RLock()
	closed := b.closed
	subs := make([]*redisSubscription, len(b.subs))
	copy(subs, b.subs)
	b.mu.RUnlock()
	if closed {
		return
	}

	for _, sub := range subs {
		if !sub.active {
			continue
		}
		if err := sub.handler(context.Background(), msg.Event); err != nil {
			log.Error().Err(err).Str("event_type", string(msg.Event.Type)).Msg("redis event handler failed")
		}
	}
}

// Close stops the subscriber goroutine and the underlying Redis connection.
func (b *RedisBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	cancel := b.cancel
	pubsub := b.pubsub
	client := b.client
	b.cancel = nil
	b.pubsub = nil
	b.client = nil
	b.mu.Unlock()

	// pubsub.Close() is what actually ends the Channel() loop.
	if pubsub != nil {
		_ = pubsub.Close()
	}
	if cancel != nil {
		cancel()
	}
	if pubsub != nil {
		<-b.done
	}
	if client != nil {
		return client.Close()
	}
	return nil
}
