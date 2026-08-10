package booktrack

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/diegobraga92/pudimproductivity/backend/internal/db"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

func setupBookTestPostgres(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("pudimproductivity"),
		postgres.WithUsername("pudim"),
		postgres.WithPassword("pudim_dev"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := db.ConnectPool(ctx, shared.DatabaseConfig{
		URL:             connStr,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("ConnectPool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return ctx, pool
}

func TestBookRepository_CRUDAndDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupBookTestPostgres(t)
	repo := NewPostgresRepository(pool)

	book, err := NewBook(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "9781250237231", "Permanent Record",
		[]string{"Edward Snowden"}, "Macmillan", "2019-09-17", "A memoir", 352, "http://img/x.jpg",
		StatusWantToRead,
	)
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}

	if err := repo.Create(ctx, book); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Duplicate ISBN → ErrDuplicateISBN.
	dup, _ := NewBook(
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "9781250237231", "Duplicate",
		nil, "", "", "", 0, "", StatusWantToRead,
	)
	if err := repo.Create(ctx, dup); !errors.Is(err, ErrDuplicateISBN) {
		t.Fatalf("want ErrDuplicateISBN, got %v", err)
	}

	// GetByISBN works.
	got, err := repo.GetByISBN(ctx, "9781250237231")
	if err != nil {
		t.Fatalf("GetByISBN: %v", err)
	}
	if got.Title != "Permanent Record" || len(got.Authors) != 1 {
		t.Fatalf("GetByISBN incomplete: %+v", got)
	}

	// UpdateStatus + List filter.
	if err := repo.UpdateStatus(ctx, book.ID, StatusReading); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	reading, err := repo.List(ctx, "reading")
	if err != nil {
		t.Fatalf("List reading: %v", err)
	}
	if len(reading) != 1 || reading[0].Status != StatusReading {
		t.Fatalf("List(reading) = %+v", reading)
	}
	unread, err := repo.List(ctx, "want_to_read")
	if err != nil {
		t.Fatalf("List want_to_read: %v", err)
	}
	if len(unread) != 0 {
		t.Fatalf("List(want_to_read) should be empty, got %d", len(unread))
	}

	// Delete.
	if err := repo.Delete(ctx, book.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, book.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID after delete: want ErrNotFound, got %v", err)
	}
}
