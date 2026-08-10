package booktrack

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const bookColumns = `id, isbn, title, authors, publisher, published_date, description, page_count, thumbnail_url, status, created_at, updated_at`

func scanBook(scanner interface{ Scan(dest ...any) error }) (*Book, error) {
	b := &Book{}
	err := scanner.Scan(
		&b.ID, &b.ISBN, &b.Title, &b.Authors,
		&b.Publisher, &b.PublishedDate, &b.Description,
		&b.PageCount, &b.ThumbnailURL, (*string)(&b.Status),
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *PostgresRepository) Create(ctx context.Context, book *Book) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO books (id, isbn, title, authors, publisher, published_date, description, page_count, thumbnail_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		book.ID, book.ISBN, book.Title, book.Authors,
		book.Publisher, book.PublishedDate, book.Description,
		book.PageCount, book.ThumbnailURL, book.Status,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrDuplicateISBN
		}
		return fmt.Errorf("insert book: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Book, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+bookColumns+` FROM books WHERE id = $1`, id)
	book, err := scanBook(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get book %s: %w", id, err)
	}
	return book, nil
}

func (r *PostgresRepository) GetByISBN(ctx context.Context, isbn string) (*Book, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+bookColumns+` FROM books WHERE isbn = $1`, isbn)
	book, err := scanBook(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get book by isbn %s: %w", isbn, err)
	}
	return book, nil
}

func (r *PostgresRepository) List(ctx context.Context, status string) ([]*Book, error) {
	query := `SELECT ` + bookColumns + ` FROM books`
	var args []any
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list books: %w", err)
	}
	defer rows.Close()

	var out []*Book
	for rows.Next() {
		book, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, book)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id string, status BookStatus) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE books SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("update book status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM books WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete book: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
