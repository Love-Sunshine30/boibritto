package auth

import (
	"context"
	"errors"
	"fmt"

	"boibritto/internal/apperror"
	"boibritto/internal/domain"
)

// FindOrCreateUser looks up a local user by Firebase UID, provisioning a new
// row on first sign-in ("just-in-time" provisioning). Firebase owns identity;
// this is where that identity gets a corresponding row in our own schema.
func (s *Store) FindOrCreateUser(ctx context.Context, firebaseUID, name, email string) (*domain.User, error) {
	user, err := s.GetUserByFirebaseUID(ctx, firebaseUID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	user, err = s.InsertUser(ctx, firebaseUID, name, email)
	if err != nil {
		return nil, fmt.Errorf("provisioning user: %w", err)
	}
	return user, nil
}
