package notification

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// Repo records which notifications have been sent so at-least-once redelivery
// from RabbitMQ does not produce duplicates. The Postgres implementation is
// durable across restarts; the memory one is for tests / degraded startups.
type Repo interface {
	// AlreadySent reports whether a notification for the event/channel pair
	// was already recorded.
	AlreadySent(ctx context.Context, eventID, channel string) (bool, error)
	// MarkSent records that the event/channel notification was delivered.
	MarkSent(ctx context.Context, eventID, channel, eventType, taskID string) error
}

// MemoryRepo is an in-memory implementation of Repo (single process only).
type MemoryRepo struct {
	mu   sync.Mutex
	sent map[string]struct{}
}

// NewMemoryRepo creates a MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{sent: make(map[string]struct{})}
}

func (m *MemoryRepo) AlreadySent(_ context.Context, eventID, channel string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.sent[eventID+"|"+channel]
	return ok, nil
}

func (m *MemoryRepo) MarkSent(_ context.Context, eventID, channel, _, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent[eventID+"|"+channel] = struct{}{}
	return nil
}

// PostgresRepo persists sent notifications in the `notifications` table. The
// UNIQUE(event_id, channel) constraint makes MarkSent idempotent across
// process restarts.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a Postgres-backed Repo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) AlreadySent(ctx context.Context, eventID, channel string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE event_id = $1 AND channel = $2`,
		eventID, channel,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostgresRepo) MarkSent(ctx context.Context, eventID, channel, eventType, taskID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notifications (id, event_id, channel, event_type, task_id)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (event_id, channel) DO NOTHING`,
		shared.NewUUID(), eventID, channel, eventType, taskID,
	)
	return err
}
