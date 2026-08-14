package library

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const itemColumns = `id, name, media_type, release_year, done, notes, score, score_source, created_at, updated_at`

func scanItem(scanner interface{ Scan(dest ...any) error }) (*Item, error) {
	it := &Item{}
	err := scanner.Scan(
		&it.ID, &it.Name, (*string)(&it.MediaType), &it.ReleaseYear, &it.Done, &it.Notes,
		&it.Score, &it.ScoreSource, &it.CreatedAt, &it.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return it, nil
}

func (r *PostgresRepository) Create(ctx context.Context, item *Item) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO library_items (id, name, media_type, release_year, done, notes, score, score_source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		item.ID, item.Name, item.MediaType, item.ReleaseYear, item.Done, item.Notes, item.Score, item.ScoreSource,
	)
	if err != nil {
		return fmt.Errorf("insert library item: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Item, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+itemColumns+` FROM library_items WHERE id = $1`, id)
	item, err := scanItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get library item %s: %w", id, err)
	}
	return item, nil
}

func (r *PostgresRepository) List(ctx context.Context, mediaType string, done *bool) ([]*Item, error) {
	query := `SELECT ` + itemColumns + ` FROM library_items`
	var args []any
	var conds []string
	if mediaType != "" {
		conds = append(conds, fmt.Sprintf(`media_type = $%d`, len(args)+1))
		args = append(args, mediaType)
	}
	if done != nil {
		conds = append(conds, fmt.Sprintf(`done = $%d`, len(args)+1))
		args = append(args, *done)
	}
	if len(conds) > 0 {
		query += ` WHERE ` + strings.Join(conds, " AND ")
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list library items: %w", err)
	}
	defer rows.Close()

	var out []*Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, item *Item) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE library_items
		SET name = $2, media_type = $3, release_year = $4, done = $5, notes = $6, score = $7, score_source = $8, updated_at = NOW()
		WHERE id = $1`,
		item.ID, item.Name, item.MediaType, item.ReleaseYear, item.Done, item.Notes, item.Score, item.ScoreSource,
	)
	if err != nil {
		return fmt.Errorf("update library item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM library_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete library item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Import persists every item in a single transaction so a failed insert rolls
// back the whole batch. Callers are expected to validate items beforehand.
func (r *PostgresRepository) Import(ctx context.Context, items []*Item) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin library import: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, item := range items {
		_, err := tx.Exec(ctx, `
			INSERT INTO library_items (id, name, media_type, release_year, done, notes, score, score_source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			item.ID, item.Name, item.MediaType, item.ReleaseYear, item.Done, item.Notes, item.Score, item.ScoreSource,
		)
		if err != nil {
			return fmt.Errorf("import library item %q: %w", item.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit library import: %w", err)
	}
	return nil
}
