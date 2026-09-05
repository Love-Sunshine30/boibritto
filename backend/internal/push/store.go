package push

import (
	"context"
	"fmt"

	"boibritto/internal/platform/postgres"
)

type Store struct {
	db postgres.Querier
}

func NewStore(db postgres.Querier) *Store {
	return &Store{db: db}
}

func (s *Store) Subscribe(ctx context.Context, userID int, platform, fcmToken string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO push_subscriptions (user_id, platform, fcm_token)
		VALUES ($1, $2, $3)
		ON CONFLICT (fcm_token) DO UPDATE
		SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform
	`, userID, platform, fcmToken)
	if err != nil {
		return fmt.Errorf("inserting push subscription: %w", err)
	}
	return nil
}

func (s *Store) Unsubscribe(ctx context.Context, fcmToken string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE fcm_token = $1`, fcmToken)
	if err != nil {
		return fmt.Errorf("deleting push subscription: %w", err)
	}
	return nil
}

// TokensForUser returns every registered device token for a user — a user
// may have multiple (web + android, or several devices).
func (s *Store) TokensForUser(ctx context.Context, userID int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT fcm_token FROM push_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying tokens: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scanning token: %w", err)
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}
