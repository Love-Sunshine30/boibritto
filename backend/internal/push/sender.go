package push

import (
	"context"
	"log/slog"

	"firebase.google.com/go/v4/messaging"
)

type Payload struct {
	Title string
	Body  string
	Data  map[string]string // arbitrary key-value data the client app can use for deep-linking
}

// Sender is the interface other domains (requests, messages) depend on —
// narrow, and deliberately doesn't expose *messaging.Client or any FCM-
// specific type, so those packages stay decoupled from the notification
// provider's implementation.
type Sender interface {
	Send(ctx context.Context, userID int, payload Payload) error
}

type FCMSender struct {
	client *messaging.Client
	store  *Store
	logger *slog.Logger
}

func NewFCMSender(client *messaging.Client, store *Store, logger *slog.Logger) *FCMSender {
	return &FCMSender{client: client, store: store, logger: logger}
}

// Send delivers to every device registered for this user. A failure on one
// device's token doesn't stop delivery to the user's other devices —
// each token's send result is handled independently.
func (s *FCMSender) Send(ctx context.Context, userID int, payload Payload) error {
	tokens, err := s.store.TokensForUser(ctx, userID)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		// Not an error — the user just has no registered devices
		// (never opened the app, or never granted notification permission).
		return nil
	}

	for _, token := range tokens {
		msg := &messaging.Message{
			Token: token,
			Notification: &messaging.Notification{
				Title: payload.Title,
				Body:  payload.Body,
			},
			Data: payload.Data,
		}

		if _, err := s.client.Send(ctx, msg); err != nil {
			// Log and continue — one bad/expired token shouldn't fail
			// delivery to this user's other devices. A token that's been
			// uninstalled/invalidated will keep failing every time, so
			// this is also where you'd eventually add auto-cleanup
			// (delete a token after N consecutive failures) — not built
			// yet, worth a follow-up once this is live for a while.
			s.logger.Error("fcm send failed", "error", err, "user_id", userID, "token_suffix", lastN(token, 8))
		}
	}
	return nil
}

func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
