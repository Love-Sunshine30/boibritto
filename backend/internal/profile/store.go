package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"boibritto/internal/apperror"
	"boibritto/internal/domain"
	"boibritto/internal/platform/postgres"
)

type Store struct {
	db postgres.Querier
}

func NewStore(db postgres.Querier) *Store {
	return &Store{db: db}
}

func (s *Store) IsProfileComplete(ctx context.Context, userID int) (bool, error) {
	var complete bool
	err := s.db.QueryRowContext(ctx, `
		SELECT whatsapp_number IS NOT NULL AND name != '' 
		FROM users WHERE id = $1
	`, userID).Scan(&complete)
	if errors.Is(err, sql.ErrNoRows) {
		return false, apperror.ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("checking profile completeness: %w", err)
	}
	return complete, nil
}

func (s *Store) UpdateProfile(ctx context.Context, userID int, name, whatsappNumber *string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users SET
			name            = COALESCE($1, name),
			whatsapp_number = COALESCE($2, whatsapp_number)
		WHERE id = $3
	`, name, whatsappNumber, userID)
	if err != nil {
		return fmt.Errorf("updating profile: %w", err)
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

func (s *Store) GetPublicProfile(ctx context.Context, userID int) (PublicUser, error) {
	var u PublicUser
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, last_active_at, created_at FROM users WHERE id = $1
	`, userID).Scan(&u.ID, &u.Name, &u.LastActiveAt, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicUser{}, apperror.ErrNotFound
	}
	if err != nil {
		return PublicUser{}, fmt.Errorf("querying public profile: %w", err)
	}
	return u, nil
}

// TouchLastActive updates last_active_at, but only if more than
// throttleWindow has passed since it was last set — avoids a write on
// every single authenticated request.
func (s *Store) TouchLastActive(ctx context.Context, userID int, throttleWindow time.Duration) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET last_active_at = now()
		WHERE id = $1
		  AND (last_active_at IS NULL OR last_active_at < now() - $2::interval)
	`, userID, fmt.Sprintf("%d seconds", int(throttleWindow.Seconds())))
	if err != nil {
		return fmt.Errorf("touching last_active_at: %w", err)
	}
	return nil
}

func (s *Store) GetUserByID(ctx context.Context, userID int) (*domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, firebase_uid, name, email, whatsapp_number, last_active_at, is_admin, created_at
		FROM users WHERE id = $1
	`, userID).Scan(&u.ID, &u.FirebaseUID, &u.Name, &u.Email, &u.WhatsAppNumber, &u.LastActiveAt, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying user by id: %w", err)
	}
	return &u, nil
}
