// internal/platform/firebase/firebase.go
package firebase

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type Clients struct {
	Auth      *auth.Client
	Messaging *messaging.Client
	Firestore *firestore.Client
}

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
	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("initializing firestore client: %w", err)
	}

	return &Clients{Auth: authClient, Messaging: messagingClient, Firestore: firestoreClient}, nil
}
