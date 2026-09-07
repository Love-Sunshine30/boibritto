package messages

import (
	"context"
	"fmt"
	"strings"

	"boibritto/internal/apperror"
	"boibritto/internal/domain"
)

// Notifier — same decoupling pattern as requests.Notifier: messages declares
// what it needs, push provides an adapter satisfying it structurally.
type Notifier interface {
	NotifyNewMessage(ctx context.Context, recipientUserID int, senderName, preview string, requestID int) error
}

// userLookup — satisfied structurally by *profile.Store, which already has
// GetUserByID. messages -> profile -> auth is a one-way chain, no cycle.
type userLookup interface {
	GetUserByID(ctx context.Context, userID int) (*domain.User, error)
}

type messageStore interface {
	CreateThread(ctx context.Context, requestID, bookID int, bookTitle string,
		ownerID int, ownerUID, ownerName string, requesterID int, requesterUID, requesterName string) error
	IsParticipant(ctx context.Context, requestID int, firebaseUID string) (bool, error)
	GetThreadForParticipant(ctx context.Context, requstID int, firebaseUid string) (threadDoc, error)
	SendMessage(ctx context.Context, requestID, senderID int, body string) error
	ListThreadsForUser(ctx context.Context, firebaseUID string) ([]threadDoc, error)
}

type Service struct {
	store    messageStore
	users    userLookup
	notifier Notifier
}

func NewService(store messageStore, users userLookup, notifier Notifier) *Service {
	return &Service{store: store, users: users, notifier: notifier}
}

// CreateThread — called by requests.Service the moment a request is
// accepted. Resolves both participants' Firebase UIDs/names itself, so
// requests never needs to know Firebase UIDs exist.
func (s *Service) CreateThread(ctx context.Context, requestID, bookID int, bookTitle string, ownerID, requesterID int) error {
	owner, err := s.users.GetUserByID(ctx, ownerID)
	if err != nil {
		return fmt.Errorf("looking up owner: %w", err)
	}
	requester, err := s.users.GetUserByID(ctx, requesterID)
	if err != nil {
		return fmt.Errorf("looking up requester: %w", err)
	}

	return s.store.CreateThread(ctx, requestID, bookID, bookTitle,
		ownerID, owner.FirebaseUID, owner.Name,
		requesterID, requester.FirebaseUID, requester.Name)
}

func (s *Service) SendMessage(ctx context.Context, requestID, senderID int, senderFirebaseUID, senderName, body string) (MessageResponse, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return MessageResponse{}, fmt.Errorf("%w: message cannot be empty", apperror.ErrValidation)
	}
	if len(body) > 2000 {
		return MessageResponse{}, fmt.Errorf("%w: message too long", apperror.ErrValidation)
	}

	isParticipant, err := s.store.IsParticipant(ctx, requestID, senderFirebaseUID)
	if err != nil {
		return MessageResponse{}, fmt.Errorf("checking participation: %w", err)
	}
	if !isParticipant {
		return MessageResponse{}, fmt.Errorf("%w: you're not a participant in this thread", apperror.ErrForbidden)
	}

	if err := s.store.SendMessage(ctx, requestID, senderID, body); err != nil {
		return MessageResponse{}, fmt.Errorf("sending message: %w", err)
	}

	threadDoc, err := s.store.GetThreadForParticipant(ctx, requestID, senderFirebaseUID)

	// get other participent ID
	recipientID := otherParticipant(threadDoc.ParticipantIDs, senderID)
	go s.notifyOtherParticipant(context.Background(), requestID, recipientID, senderName, body)

	return MessageResponse{SenderID: senderID, Body: body}, nil
}

func otherParticipant(ids []int, senderID int) int {
	for _, id := range ids {
		if id != senderID {
			return id
		}
	}
	return 0 // shouldn't happen for a 2-person thread; worth logging if it does
}

func (s *Service) notifyOtherParticipant(ctx context.Context, requestID int, recipientID int, senderName, body string) {
	// Implementation detail: needs the other participant's internal ID,
	// which requires exposing it from the store. Kept simple here —
	// a fuller version would have IsParticipant's underlying lookup
	// also return both participant IDs so this doesn't re-fetch.
	_ = s.notifier.NotifyNewMessage(ctx, recipientID, senderName, body, requestID)
}

func (s *Service) ListThreads(ctx context.Context, firebaseUID string) ([]ThreadSummary, error) {
	threads, err := s.store.ListThreadsForUser(ctx, firebaseUID)
	if err != nil {
		return nil, fmt.Errorf("listing threads: %w", err)
	}
	// Response mapping omitted for brevity — pick the "other" participant's
	// name from ParticipantNames based on which UID isn't the caller.
	summaries := make([]ThreadSummary, 0, len(threads))
	for _, t := range threads {
		summaries = append(summaries, ThreadSummary{
			RequestID: t.RequestID, BookTitle: t.BookTitle,
			LastMessagePreview: t.LastMessagePreview, LastMessageAt: t.LastMessageAt,
		})
	}
	return summaries, nil
}
