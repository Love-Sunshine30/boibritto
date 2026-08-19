package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"boibritto/internal/apperror"
	"boibritto/internal/domain"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetUserByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, firebase_uid, name, email, whatsapp_number, is_admin, created_at
		FROM users
		WHERE firebase_uid = $1
	`, firebaseUID).Scan(&u.ID, &u.FirebaseUID, &u.Name, &u.Email, &u.WhatsAppNumber, &u.IsAdmin, &u.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying user by firebase uid: %w", err)
	}
	return &u, nil
}

func (s *Store) InsertUser(ctx context.Context, firebaseUID, name, email string) (*domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (firebase_uid, name, email)
		VALUES ($1, $2, $3)
		RETURNING id, firebase_uid, name, email, whatsapp_number, is_admin, created_at
	`, firebaseUID, name, email).Scan(&u.ID, &u.FirebaseUID, &u.Name, &u.Email, &u.WhatsAppNumber, &u.IsAdmin, &u.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}
	return &u, nil
}
