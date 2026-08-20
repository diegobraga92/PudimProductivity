package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
)

type LibraryRepository struct {
	pool *pgxpool.Pool
}

func NewLibraryRepository(pool *pgxpool.Pool) *LibraryRepository {
	return &LibraryRepository{pool: pool}
}

const itemColumns = `id, name, media_type, release_year, done, notes, subtype, score, score_source, created_at, updated_at`

func scanItem(scanner interface{ Scan(dest ...any) error }) (*library.Item, error) {
	it := &library.Item{}
	err := scanner.Scan(
		&it.ID, &it.Name, (*string)(&it.MediaType), &it.ReleaseYear, &it.Done, &it.Notes,
		&it.Subtype, &it.Score, &it.ScoreSource, &it.CreatedAt, &it.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return it, nil
}

func (r *LibraryRepository) Create(ctx context.Context, item *library.Item) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO library_items (id, name, media_type, release_year, done, notes, subtype, score, score_source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		item.ID, item.Name, item.MediaType, item.ReleaseYear, item.Done, item.Notes, item.Subtype, item.Score, item.ScoreSource,
	)
	if err != nil {
		return fmt.Errorf("insert library item: %w", err)
	}
	return nil
}

func (r *LibraryRepository) GetByID(ctx context.Context, id string) (*library.Item, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+itemColumns+` FROM library_items WHERE id = $1`, id)
	item, err := scanItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, library.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get library item %s: %w", id, err)
	}
	return item, nil
}

func (r *LibraryRepository) List(ctx context.Context, filter library.ListFilter) ([]*library.Item, error) {
	query := `SELECT ` + itemColumns + ` FROM library_items`
	var args []any
	var conds []string
	if filter.MediaType != "" {
		conds = append(conds, fmt.Sprintf(`media_type = $%d`, len(args)+1))
		args = append(args, filter.MediaType)
	}
	if filter.Done != nil {
		conds = append(conds, fmt.Sprintf(`done = $%d`, len(args)+1))
		args = append(args, *filter.Done)
	}
	if filter.Subtype != "" {
		conds = append(conds, fmt.Sprintf(`LOWER(subtype) = LOWER($%d)`, len(args)+1))
		args = append(args, filter.Subtype)
	}
	if len(conds) > 0 {
		query += ` WHERE ` + strings.Join(conds, ` AND `)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list library items: %w", err)
	}
	defer rows.Close()

	var out []*library.Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *LibraryRepository) DistinctSubtypes(ctx context.Context, mediaType string) ([]string, error) {
	query := `SELECT DISTINCT subtype FROM library_items WHERE subtype <> ''`
	args := []any{}
	if mediaType != "" {
		query += ` AND media_type = $1`
		args = append(args, mediaType)
	}
	query += ` ORDER BY subtype ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("distinct library subtypes: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = make([]string, 0)
	}
	return out, nil
}

func (r *LibraryRepository) Update(ctx context.Context, item *library.Item) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE library_items
		SET name = $2, media_type = $3, release_year = $4, done = $5, notes = $6, subtype = $7, score = $8, score_source = $9, updated_at = NOW()
		WHERE id = $1`,
		item.ID, item.Name, item.MediaType, item.ReleaseYear, item.Done, item.Notes, item.Subtype, item.Score, item.ScoreSource,
	)
	if err != nil {
		return fmt.Errorf("update library item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return library.ErrNotFound
	}
	return nil
}

func (r *LibraryRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM library_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete library item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return library.ErrNotFound
	}
	return nil
}

// Import persists every item in a single transaction so a failed insert rolls
// back the whole batch. Callers are expected to validate items beforehand.
func (r *LibraryRepository) Import(ctx context.Context, items []*library.Item) error {
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
			INSERT INTO library_items (id, name, media_type, release_year, done, notes, subtype, score, score_source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			item.ID, item.Name, item.MediaType, item.ReleaseYear, item.Done, item.Notes, item.Subtype, item.Score, item.ScoreSource,
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
