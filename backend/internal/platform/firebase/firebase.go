package firebase

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type Clients struct {
	Auth      *auth.Client
	Messaging *messaging.Client
}

// New initializes the Firebase Admin SDK from raw service account JSON
// (provided via env var, not a file path). WithAuthCredentialsJSON with
// option.ServiceAccount is used instead of the deprecated
// WithCredentialsJSON, so the SDK validates that this is actually a
// service account credential before using it, rather than trusting an
// unvalidated blob.
func New(ctx context.Context, projectID, serviceAccountJSON string) (*Clients, error) {
	opt := option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(serviceAccountJSON))

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID}, opt)
	if err != nil {
		return nil, fmt.Errorf("initializing firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initializing firebase auth client: %w", err)
	}

	messagingClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("initializing firebase messaging client: %w", err)
	}

	return &Clients{Auth: authClient, Messaging: messagingClient}, nil
}
