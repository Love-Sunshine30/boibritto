package requests

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"boibritto/internal/apperror"
	"boibritto/internal/platform/postgres"
)

type requestStore interface {
	GetRequestByID(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error)
	HasPendingRequest(ctx context.Context, bookID, requesterID int) (bool, error)
	InsertRequest(ctx context.Context, bookID, requesterID int, message string) (BorrowRequest, error)
	ListSent(ctx context.Context, requesterID int) ([]BorrowRequest, error)
	ListIncoming(ctx context.Context, ownerID int) ([]BorrowRequest, error)
	UpdateStatus(ctx context.Context, q postgres.Querier, id int, status Status) error
	SetOwnerConfirmed(ctx context.Context, q postgres.Querier, id int, confirmed bool) error
	SetBorrowerConfirmed(ctx context.Context, q postgres.Querier, id int, confirmed bool) error
}

type bookStore interface {
	SetAvailability(ctx context.Context, q postgres.Querier, bookID int, available bool) error
}

type Service struct {
	db        *sql.DB
	store     requestStore
	bookStore bookStore
	notifier  Notifier
	logger    *slog.Logger
}

func NewService(db *sql.DB, store requestStore, bookStore bookStore, notifier Notifier, logger *slog.Logger) *Service {
	return &Service{db: db, store: store, bookStore: bookStore, notifier: notifier, logger: logger}
}

func (s *Service) CreateRequest(ctx context.Context, bookID, requesterID, bookOwnerID int, message string) (BorrowRequestResponse, error) {
	if requesterID == bookOwnerID {
		return BorrowRequestResponse{}, fmt.Errorf("%w: you can't request your own book", apperror.ErrValidation)
	}

	if len(message) > 500 {
		return BorrowRequestResponse{}, fmt.Errorf("%w: message too long", apperror.ErrValidation)
	}

	pending, err := s.store.HasPendingRequest(ctx, bookID, requesterID)
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("checking pending request: %w", err)
	}
	if pending {
		return BorrowRequestResponse{}, fmt.Errorf("%w: you already have a pending request for this book", apperror.ErrConflict)
	}
	r, err := s.store.InsertRequest(ctx, bookID, requesterID, message)
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("creating request: %w", err)
	}
	return toResponse(r), nil
}

func (s *Service) ListSent(ctx context.Context, requesterID int) ([]BorrowRequestResponse, error) {
	reqs, err := s.store.ListSent(ctx, requesterID)
	if err != nil {
		return nil, fmt.Errorf("listing sent requests: %w", err)
	}
	return toResponses(reqs), nil
}

func (s *Service) ListIncoming(ctx context.Context, ownerID int) ([]BorrowRequestResponse, error) {
	reqs, err := s.store.ListIncoming(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("listing incoming requests: %w", err)
	}
	return toResponses(reqs), nil
}

// UpdateStatus handles accept/reject — owner-only, only valid from pending.
func (s *Service) UpdateStatus(ctx context.Context, requestID, ownerID int, newStatus Status) (BorrowRequestResponse, error) {
	if newStatus != StatusAccepted && newStatus != StatusRejected {
		return BorrowRequestResponse{}, fmt.Errorf("%w: status must be accepted or rejected", apperror.ErrValidation)
	}

	existing, err := s.store.GetRequestByID(ctx, s.db, requestID)
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("looking up request: %w", err)
	}
	if existing.OwnerID != ownerID {
		return BorrowRequestResponse{}, fmt.Errorf("%w: only the book's owner can accept or reject requests", apperror.ErrForbidden)
	}
	if existing.Status != StatusPending {
		return BorrowRequestResponse{}, fmt.Errorf("%w: request is not pending", apperror.ErrValidation)
	}

	if err := s.store.UpdateStatus(ctx, s.db, requestID, newStatus); err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("updating request: %w", err)
	}
	updated, err := s.store.GetRequestByID(ctx, s.db, requestID)
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("re-fetching request: %w", err)
	}
	return toResponse(updated), nil
}

// ConfirmHandoff records the caller's confirmation (owner or borrower —
// determined from who they are relative to the request, not a client-
// supplied role). Once BOTH confirmations are in, the request transitions
// to active and the book becomes unavailable, atomically. If only this
// caller has confirmed so far, the other party is notified (best-effort).
func (s *Service) ConfirmHandoff(ctx context.Context, requestID, callerID int) (BorrowRequestResponse, error) {
	existing, err := s.store.GetRequestByID(ctx, s.db, requestID)
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("looking up request: %w", err)
	}
	if existing.Status != StatusAccepted {
		return BorrowRequestResponse{}, fmt.Errorf("%w: request must be accepted before confirming handoff", apperror.ErrValidation)
	}

	isOwner := callerID == existing.OwnerID
	isBorrower := callerID == existing.RequesterID
	if !isOwner && !isBorrower {
		return BorrowRequestResponse{}, fmt.Errorf("%w: you're not a participant in this request", apperror.ErrForbidden)
	}

	var otherPartyID int
	var willBothBeConfirmed bool
	err = postgres.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if isOwner {
			if err := s.store.SetOwnerConfirmed(ctx, tx, requestID, true); err != nil {
				return err
			}
			otherPartyID = existing.RequesterID
			willBothBeConfirmed = existing.BorrowerConfirmed // the other side, already true?
		} else {
			if err := s.store.SetBorrowerConfirmed(ctx, tx, requestID, true); err != nil {
				return err
			}
			otherPartyID = existing.OwnerID
			willBothBeConfirmed = existing.OwnerConfirmed
		}

		if willBothBeConfirmed {
			if err := s.store.UpdateStatus(ctx, tx, requestID, StatusActive); err != nil {
				return fmt.Errorf("activating request: %w", err)
			}
			if err := s.bookStore.SetAvailability(ctx, tx, existing.BookID, false); err != nil {
				return fmt.Errorf("marking book unavailable: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("confirming handoff: %w", err)
	}

	if !willBothBeConfirmed {
		// Best-effort — a failed notification shouldn't fail the request
		// that already succeeded. Logged, not propagated.
		if err := s.notifier.NotifyHandoffConfirmationNeeded(ctx, otherPartyID, requestID); err != nil {
			s.logger.Error("failed to notify handoff confirmation", "error", err, "request_id", requestID, "recipient_id", otherPartyID)
		}
	}

	updated, err := s.store.GetRequestByID(ctx, s.db, requestID)
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("re-fetching request: %w", err)
	}
	return toResponse(updated), nil
}

// MarkReturned — owner-only, only valid from active.
func (s *Service) MarkReturned(ctx context.Context, requestID, ownerID int) (BorrowRequestResponse, error) {
	existing, err := s.store.GetRequestByID(ctx, s.db, requestID)
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("looking up request: %w", err)
	}
	if existing.OwnerID != ownerID {
		return BorrowRequestResponse{}, fmt.Errorf("%w: only the book's owner can mark it returned", apperror.ErrForbidden)
	}
	if existing.Status != StatusActive {
		return BorrowRequestResponse{}, fmt.Errorf("%w: request is not currently active", apperror.ErrValidation)
	}

	err = postgres.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.store.UpdateStatus(ctx, tx, requestID, StatusReturned); err != nil {
			return err
		}
		return s.bookStore.SetAvailability(ctx, tx, existing.BookID, true)
	})
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("marking returned: %w", err)
	}

	updated, err := s.store.GetRequestByID(ctx, s.db, requestID)
	if err != nil {
		return BorrowRequestResponse{}, fmt.Errorf("re-fetching request: %w", err)
	}
	return toResponse(updated), nil
}
