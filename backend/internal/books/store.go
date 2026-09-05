package books

import (
	"boibritto/internal/apperror"
	"boibritto/internal/platform/postgres"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Book mirrors the DB row + a joined owner name. Kept separate from
// BookResponse (dto.go) deliberately — this is the store's internal shape.
type Book struct {
	ID          int
	Title       string
	Author      string
	Genre       string
	Description string
	CoverURL    string
	Available   bool
	OwnerID     int
	OwnerName   string
	CreatedAt   time.Time
}

type Store struct {
	db postgres.Querier
}

func NewStore(db postgres.Querier) *Store {
	return &Store{db: db}
}

const pageSize = 20

func (s *Store) ListBooks(ctx context.Context, cursor *time.Time) ([]Book, error) {
	query := `
		SELECT b.id, b.title, b.author, b.genre, b.description, b.cover_url,
		       b.available, b.owner_id, u.name, b.created_at
		FROM books b
		JOIN users u ON u.id = b.owner_id
		WHERE ($1::timestamptz IS NULL OR b.created_at < $1)
		ORDER BY b.created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, cursor, pageSize)
	if err != nil {
		return nil, fmt.Errorf("querying books: %w", err)
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.Description,
			&b.CoverURL, &b.Available, &b.OwnerID, &b.OwnerName, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning book row: %w", err)
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating book rows: %w", err)
	}
	return books, nil
}

func (s *Store) InsertBook(ctx context.Context, ownerID int, req CreateBookRequest) (Book, error) {
	var b Book
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO books (title, author, genre, description, cover_url, available, owner_id, created_at)
		VALUES ($1, $2, $3, $4, $5, true, $6, now())
		RETURNING id, title, author, genre, description, cover_url, available, owner_id, created_at
	`, req.Title, req.Author, req.Genre, req.Description, req.CoverURL, ownerID).Scan(
		&b.ID, &b.Title, &b.Author, &b.Genre, &b.Description, &b.CoverURL, &b.Available, &b.OwnerID, &b.CreatedAt,
	)
	if err != nil {
		return Book{}, fmt.Errorf("inserting book: %w", err)
	}
	return b, nil
}

func (s *Store) GetBookByID(ctx context.Context, id int) (Book, error) {
	var b Book
	err := s.db.QueryRowContext(ctx, `
		SELECT b.id, b.title, b.author, b.genre, b.description, b.cover_url,
		       b.available, b.owner_id, u.name, b.created_at
		FROM books b
		JOIN users u ON u.id = b.owner_id
		WHERE b.id = $1
	`, id).Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.Description,
		&b.CoverURL, &b.Available, &b.OwnerID, &b.OwnerName, &b.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return Book{}, apperror.ErrNotFound
	}
	if err != nil {
		return Book{}, fmt.Errorf("querying book by id: %w", err)
	}
	return b, nil
}

func (s *Store) UpdateBook(ctx context.Context, id int, req UpdateBookRequest) (Book, error) {
	var b Book
	err := s.db.QueryRowContext(ctx, `
		UPDATE books SET
			title       = COALESCE($1, title),
			author      = COALESCE($2, author),
			genre       = COALESCE($3, genre),
			description = COALESCE($4, description),
			cover_url   = COALESCE($5, cover_url),
			available   = COALESCE($6, available)
		WHERE id = $7
		RETURNING id, title, author, genre, description, cover_url, available, owner_id, created_at
	`, req.Title, req.Author, req.Genre, req.Description, req.CoverURL, req.Available, id).Scan(
		&b.ID, &b.Title, &b.Author, &b.Genre, &b.Description, &b.CoverURL, &b.Available, &b.OwnerID, &b.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Book{}, apperror.ErrNotFound
	}
	if err != nil {
		return Book{}, fmt.Errorf("updating book: %w", err)
	}
	return b, nil
}

func (s *Store) DeleteBook(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM books WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting book: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking delete result: %w", err)
	}
	if rows == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (s *Store) SetAvailability(ctx context.Context, q postgres.Querier, bookID int, available bool) error {
	result, err := q.ExecContext(ctx, `UPDATE books SET available = $1 WHERE id = $2`, available, bookID)
	if err != nil {
		return fmt.Errorf("updating book availability: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking update result: %w", err)
	}

	if rows == 0 {
		return apperror.ErrNotFound
	}

	return nil
}

func (s *Store) ListByOwner(ctx context.Context, ownerID, limit int) ([]Book, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.title, b.author, b.genre, b.description, b.cover_url,
		       b.available, b.owner_id, u.name, b.created_at
		FROM books b
		JOIN users u ON u.id = b.owner_id
		WHERE b.owner_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2
	`, ownerID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying books by owner: %w", err)
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.Description,
			&b.CoverURL, &b.Available, &b.OwnerID, &b.OwnerName, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning book: %w", err)
		}
		books = append(books, b)
	}
	return books, rows.Err()
}
