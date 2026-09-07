package messages

import (
	"context"
	"fmt"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"boibritto/internal/apperror"
)

type Store struct {
	client *firestore.Client
}

func NewStore(client *firestore.Client) *Store {
	return &Store{client: client}
}

type threadDoc struct {
	RequestID          int               `firestore:"requestId"`
	BookID             int               `firestore:"bookId"`
	BookTitle          string            `firestore:"bookTitle"`
	ParticipantIDs     []int             `firestore:"participantIds"`   // internal user IDs — backend convenience
	ParticipantUIDs    []string          `firestore:"participantUids"`  // Firebase UIDs — what Firestore security rules check
	ParticipantNames   map[string]string `firestore:"participantNames"` // keyed by internal ID as string
	LastMessagePreview string            `firestore:"lastMessagePreview"`
	LastMessageAt      time.Time         `firestore:"lastMessageAt"`
	CreatedAt          time.Time         `firestore:"createdAt"`
}

type messageDoc struct {
	SenderID  int       `firestore:"senderId"`
	Body      string    `firestore:"body"`
	CreatedAt time.Time `firestore:"createdAt"`
}

func threadDocID(requestID int) string {
	return fmt.Sprintf("%d", requestID)
}

func (s *Store) CreateThread(ctx context.Context, requestID, bookID int, bookTitle string,
	ownerID int, ownerUID, ownerName string, requesterID int, requesterUID, requesterName string) error {

	_, err := s.client.Collection("threads").Doc(threadDocID(requestID)).Set(ctx, threadDoc{
		RequestID:       requestID,
		BookID:          bookID,
		BookTitle:       bookTitle,
		ParticipantIDs:  []int{ownerID, requesterID},
		ParticipantUIDs: []string{ownerUID, requesterUID},
		ParticipantNames: map[string]string{
			fmt.Sprintf("%d", ownerID):     ownerName,
			fmt.Sprintf("%d", requesterID): requesterName,
		},
		CreatedAt:     time.Now(),
		LastMessageAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("creating thread: %w", err)
	}
	return nil
}

func (s *Store) getThreadDoc(ctx context.Context, requestID int) (threadDoc, error) {
	snap, err := s.client.Collection("threads").Doc(threadDocID(requestID)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return threadDoc{}, apperror.ErrNotFound
		}
		return threadDoc{}, fmt.Errorf("getting thread: %w", err)
	}
	var t threadDoc
	if err := snap.DataTo(&t); err != nil {
		return threadDoc{}, fmt.Errorf("decoding thread: %w", err)
	}
	return t, nil
}

// IsParticipant checks membership by Firebase UID — the same check
// Firestore security rules apply client-side, kept consistent here for the
// REST API path.
func (s *Store) IsParticipant(ctx context.Context, requestID int, firebaseUID string) (bool, error) {
	t, err := s.getThreadDoc(ctx, requestID)
	if err != nil {
		return false, err
	}

	if slices.Contains(t.ParticipantUIDs, firebaseUID) {
		return true, nil
	}

	return false, nil
}

func (s *Store) GetThreadForParticipant(ctx context.Context, requestID int, firebaseUID string) (threadDoc, error) {
	t, err := s.getThreadDoc(ctx, requestID)
	if err != nil {
		return threadDoc{}, err
	}
	if !slices.Contains(t.ParticipantUIDs, firebaseUID) {
		return threadDoc{}, apperror.ErrForbidden
	}
	return t, nil
}

func (s *Store) SendMessage(ctx context.Context, requestID, senderID int, body string) error {
	threadRef := s.client.Collection("threads").Doc(threadDocID(requestID))

	_, _, err := threadRef.Collection("messages").Add(ctx, messageDoc{
		SenderID: senderID, Body: body, CreatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("adding message: %w", err)
	}

	_, err = threadRef.Set(ctx, map[string]any{
		"lastMessagePreview": body,
		"lastMessageAt":      time.Now(),
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("updating thread preview: %w", err)
	}
	return nil
}

func (s *Store) ListThreadsForUser(ctx context.Context, firebaseUID string) ([]threadDoc, error) {
	iter := s.client.Collection("threads").
		Where("participantUids", "array-contains", firebaseUID).
		OrderBy("lastMessageAt", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var threads []threadDoc
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing threads: %w", err)
		}
		var t threadDoc
		if err := doc.DataTo(&t); err != nil {
			return nil, fmt.Errorf("decoding thread: %w", err)
		}
		threads = append(threads, t)
	}
	return threads, nil
}
