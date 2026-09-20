package forum

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"boibritto/internal/apperror"
	"boibritto/internal/platform/postgres"
)

const pageSize = 20

type postRow struct {
	ID        int
	BookID    int
	UserID    int
	UserName  string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Store struct {
	db postgres.Querier
}

func NewStore(db postgres.Querier) *Store {
	return &Store{db: db}
}

func (s *Store) ListByBook(ctx context.Context, bookID int, cursor *time.Time) ([]postRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.book_id, p.user_id, u.name, p.body, p.created_at, p.updated_at
		FROM book_forum_posts p
		JOIN users u ON u.id = p.user_id
		WHERE p.book_id = $1 AND ($2::timestamptz IS NULL OR p.created_at < $2)
		ORDER BY p.created_at DESC
		LIMIT $3
	`, bookID, cursor, pageSize)
	if err != nil {
		return nil, fmt.Errorf("querying forum posts: %w", err)
	}
	defer rows.Close()

	var posts []postRow
	for rows.Next() {
		var p postRow
		if err := rows.Scan(&p.ID, &p.BookID, &p.UserID, &p.UserName, &p.Body, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning forum post: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func (s *Store) GetByID(ctx context.Context, id int) (postRow, error) {
	var p postRow
	err := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.book_id, p.user_id, u.name, p.body, p.created_at, p.updated_at
		FROM book_forum_posts p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = $1
	`, id).Scan(&p.ID, &p.BookID, &p.UserID, &p.UserName, &p.Body, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return postRow{}, apperror.ErrNotFound
	}
	if err != nil {
		return postRow{}, fmt.Errorf("querying forum post: %w", err)
	}
	return p, nil
}

func (s *Store) Insert(ctx context.Context, bookID, userID int, body string) (postRow, error) {
	var id int
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO book_forum_posts (book_id, user_id, body)
		VALUES ($1, $2, $3)
		RETURNING id
	`, bookID, userID, body).Scan(&id)
	if err != nil {
		return postRow{}, fmt.Errorf("inserting forum post: %w", err)
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int, body string) (postRow, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE book_forum_posts SET body = $1, updated_at = now() WHERE id = $2
	`, body, id)
	if err != nil {
		return postRow{}, fmt.Errorf("updating forum post: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return postRow{}, fmt.Errorf("checking update result: %w", err)
	}
	if rows == 0 {
		return postRow{}, apperror.ErrNotFound
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM book_forum_posts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting forum post: %w", err)
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
