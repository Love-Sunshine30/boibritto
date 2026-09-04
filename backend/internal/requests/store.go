package requests

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"boibritto/internal/apperror"
	"boibritto/internal/platform/postgres"
)

type BorrowRequest struct {
	ID                int
	BookID            int
	BookTitle         string
	RequesterID       int
	RequesterName     string
	Message           string
	OwnerID           int
	Status            Status
	OwnerConfirmed    bool
	BorrowerConfirmed bool
	CreatedAt         time.Time
}

type Store struct {
	db postgres.Querier
}

func NewStore(db postgres.Querier) *Store {
	return &Store{db: db}
}

const selectBorrowRequest = `
	SELECT br.id, br.book_id, b.title, b.owner_id,
	       br.requester_id, u.name, br.status, br.message,
	       br.owner_confirmed, br.borrower_confirmed, br.created_at
	FROM borrow_requests br
	JOIN books b ON b.id = br.book_id
	JOIN users u ON u.id = br.requester_id
`

func scanBorrowRequest(row interface{ Scan(dest ...any) error }) (BorrowRequest, error) {
	var r BorrowRequest
	err := row.Scan(&r.ID, &r.BookID, &r.BookTitle, &r.OwnerID,
		&r.RequesterID, &r.RequesterName, &r.Status, &r.Message,
		&r.OwnerConfirmed, &r.BorrowerConfirmed, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return BorrowRequest{}, apperror.ErrNotFound
	}
	if err != nil {
		return BorrowRequest{}, fmt.Errorf("scanning borrow request: %w", err)
	}
	return r, nil
}

func (s *Store) GetRequestByID(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error) {
	return scanBorrowRequest(q.QueryRowContext(ctx, selectBorrowRequest+` WHERE br.id = $1`, id))
}

func (s *Store) HasPendingRequest(ctx context.Context, bookID, requesterID int) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM borrow_requests
			WHERE book_id = $1 AND requester_id = $2 AND status = $3
		)
	`, bookID, requesterID, StatusPending).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking pending request: %w", err)
	}
	return exists, nil
}

func (s *Store) InsertRequest(ctx context.Context, bookID, requesterID int, message string) (BorrowRequest, error) {
	var id int
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO borrow_requests (book_id, requester_id, status, message, created_at)
		VALUES ($1, $2, $3, $4, now())
		RETURNING id
	`, bookID, requesterID, StatusPending, message).Scan(&id)
	if err != nil {
		return BorrowRequest{}, fmt.Errorf("inserting borrow request: %w", err)
	}
	return s.GetRequestByID(ctx, s.db, id)
}

func (s *Store) ListSent(ctx context.Context, requesterID int) ([]BorrowRequest, error) {
	return s.listWhere(ctx, `WHERE br.requester_id = $1 ORDER BY br.created_at DESC`, requesterID)
}

func (s *Store) ListIncoming(ctx context.Context, ownerID int) ([]BorrowRequest, error) {
	return s.listWhere(ctx, `WHERE b.owner_id = $1 ORDER BY br.created_at DESC`, ownerID)
}

func (s *Store) listWhere(ctx context.Context, whereClause string, arg int) ([]BorrowRequest, error) {
	rows, err := s.db.QueryContext(ctx, selectBorrowRequest+whereClause, arg)
	if err != nil {
		return nil, fmt.Errorf("querying borrow requests: %w", err)
	}
	defer rows.Close()

	var reqs []BorrowRequest
	for rows.Next() {
		r, err := scanBorrowRequest(rows)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, r)
	}
	return reqs, rows.Err()
}

func (s *Store) UpdateStatus(ctx context.Context, q postgres.Querier, id int, status Status) error {
	return s.execUpdate(ctx, q, `UPDATE borrow_requests SET status = $1 WHERE id = $2`, status, id)
}

// SetOwnerConfirmed and SetBorrowerConfirmed are separate, narrow methods
// (rather than one generic "set a column by name") so there's no risk of
// building a SQL column name from user input.
func (s *Store) SetOwnerConfirmed(ctx context.Context, q postgres.Querier, id int, confirmed bool) error {
	return s.execUpdate(ctx, q, `UPDATE borrow_requests SET owner_confirmed = $1 WHERE id = $2`, confirmed, id)
}

func (s *Store) SetBorrowerConfirmed(ctx context.Context, q postgres.Querier, id int, confirmed bool) error {
	return s.execUpdate(ctx, q, `UPDATE borrow_requests SET borrower_confirmed = $1 WHERE id = $2`, confirmed, id)
}

func (s *Store) execUpdate(ctx context.Context, q postgres.Querier, query string, args ...any) error {
	result, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing update: %w", err)
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
