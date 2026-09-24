package requests

import (
	"context"
	"errors"
	"testing"

	"boibritto/internal/apperror"
	"boibritto/internal/books"
	"boibritto/internal/platform/postgres"
)

type fakeReqStore struct {
	getFn                  func(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error)
	hasPendingFn           func(ctx context.Context, bookID, requesterID int) (bool, error)
	insertFn               func(ctx context.Context, bookID, requesterID int, message string) (BorrowRequest, error)
	updateStatusFn         func(ctx context.Context, q postgres.Querier, id int, status Status) error
	setOwnerConfirmedFn    func(ctx context.Context, q postgres.Querier, id int, confirmed bool) error
	setBorrowerConfirmedFn func(ctx context.Context, q postgres.Querier, id int, confirmed bool) error
}

func (f *fakeReqStore) GetRequestByID(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error) {
	return f.getFn(ctx, q, id)
}
func (f *fakeReqStore) HasPendingRequest(ctx context.Context, bookID, requesterID int) (bool, error) {
	return f.hasPendingFn(ctx, bookID, requesterID)
}
func (f *fakeReqStore) InsertRequest(ctx context.Context, bookID, requesterID int, message string) (BorrowRequest, error) {
	return f.insertFn(ctx, bookID, requesterID, message)
}
func (f *fakeReqStore) ListSent(ctx context.Context, id int) ([]BorrowRequest, error) {
	return nil, nil
}
func (f *fakeReqStore) ListIncoming(ctx context.Context, id int) ([]BorrowRequest, error) {
	return nil, nil
}
func (f *fakeReqStore) UpdateStatus(ctx context.Context, q postgres.Querier, id int, status Status) error {
	return f.updateStatusFn(ctx, q, id, status)
}
func (f *fakeReqStore) SetOwnerConfirmed(ctx context.Context, q postgres.Querier, id int, c bool) error {
	return f.setOwnerConfirmedFn(ctx, q, id, c)
}
func (f *fakeReqStore) SetBorrowerConfirmed(ctx context.Context, q postgres.Querier, id int, c bool) error {
	return f.setBorrowerConfirmedFn(ctx, q, id, c)
}

type fakeBookStore struct {
	setAvailabilityFn func(ctx context.Context, q postgres.Querier, bookID int, available bool) error
	getbookFn         func(ctx context.Context, bookID int) (books.Book, error)
}

func (f *fakeBookStore) SetAvailability(ctx context.Context, q postgres.Querier, bookID int, available bool) error {
	return f.setAvailabilityFn(ctx, q, bookID, available)
}

func (f *fakeBookStore) GetBookByID(ctx context.Context, bookID int) (books.Book, error) {
	return f.getbookFn(ctx, bookID)
}

func noopSetAvailability(ctx context.Context, q postgres.Querier, bookID int, available bool) error {
	return nil
}

type fakeProfileChecker struct{ complete bool }

func (f *fakeProfileChecker) IsProfileComplete(ctx context.Context, userID int) (bool, error) {
	return f.complete, nil
}

type fakeThreadCreator struct {
	called bool
}

func (f *fakeThreadCreator) CreateThread(ctx context.Context, requestID, bookID int, bookTitle string, ownerID, requesterID int) error {
	f.called = true
	return nil
}

// newTestService is a small constructor helper so every test doesn't need
// to repeat all seven NewService arguments — pass only the fakes a given
// test actually cares about, everything else gets a harmless default.
func newTestService(store requestStore, bookStore bookStore) *Service {
	return NewService(nil, store, bookStore, NoopNotifier{}, &fakeProfileChecker{complete: true}, &fakeThreadCreator{}, nil)
}

// Note: WithTx requires a real *sql.DB to open a transaction on. For pure
// unit tests of the decision logic (not the transaction plumbing itself),
// these tests focus on the pre-transaction validation/authorization checks,
// which is where the actual business rules live. Testing WithTx's own
// commit/rollback behavior is better suited to an integration test against
// a real test database.

func TestCreateRequest_RejectsSelfRequest(t *testing.T) {
	svc := newTestService(&fakeReqStore{}, &fakeBookStore{})

	_, err := svc.CreateRequest(context.Background(), 1, 5, 5, "")

	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation for self-request, got %v", err)
	}
}

func TestCreateRequest_RejectsDuplicatePending(t *testing.T) {
	store := &fakeReqStore{
		hasPendingFn: func(ctx context.Context, bookID, requesterID int) (bool, error) { return true, nil },
	}
	svc := newTestService(store, &fakeBookStore{})

	_, err := svc.CreateRequest(context.Background(), 1, 2, 5, "")

	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("expected ErrConflict for duplicate pending request, got %v", err)
	}
}

func TestCreateRequest_RejectsOverlongMessage(t *testing.T) {
	svc := newTestService(&fakeReqStore{}, &fakeBookStore{})

	longMsg := make([]byte, 501)
	_, err := svc.CreateRequest(context.Background(), 1, 2, 5, string(longMsg))

	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation for overlong message, got %v", err)
	}
}

func TestUpdateStatus_RejectsNonOwner(t *testing.T) {
	store := &fakeReqStore{
		getFn: func(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error) {
			return BorrowRequest{ID: id, OwnerID: 1, Status: StatusPending}, nil
		},
	}
	svc := newTestService(store, &fakeBookStore{})

	_, err := svc.UpdateStatus(context.Background(), 1, 2, StatusAccepted)

	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-owner accept/reject, got %v", err)
	}
}

func TestUpdateStatus_RejectsInvalidTargetStatus(t *testing.T) {
	svc := newTestService(&fakeReqStore{}, &fakeBookStore{})

	_, err := svc.UpdateStatus(context.Background(), 1, 1, StatusActive)

	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation for invalid target status, got %v", err)
	}
}

func TestUpdateStatus_RejectsWhenNotPending(t *testing.T) {
	store := &fakeReqStore{
		getFn: func(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error) {
			return BorrowRequest{ID: id, OwnerID: 1, Status: StatusAccepted}, nil
		},
	}
	svc := newTestService(store, &fakeBookStore{})

	_, err := svc.UpdateStatus(context.Background(), 1, 1, StatusAccepted)

	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation for accepting an already-accepted request, got %v", err)
	}
}

func TestConfirmHandoff_RejectsNonParticipant(t *testing.T) {
	store := &fakeReqStore{
		getFn: func(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error) {
			return BorrowRequest{ID: id, OwnerID: 1, RequesterID: 2, Status: StatusAccepted}, nil
		},
	}
	svc := newTestService(store, &fakeBookStore{})

	_, err := svc.ConfirmHandoff(context.Background(), 1, 99)

	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-participant confirm, got %v", err)
	}
}

func TestConfirmHandoff_RejectsBeforeAccepted(t *testing.T) {
	store := &fakeReqStore{
		getFn: func(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error) {
			return BorrowRequest{ID: id, OwnerID: 1, RequesterID: 2, Status: StatusPending}, nil
		},
	}
	svc := newTestService(store, &fakeBookStore{})

	_, err := svc.ConfirmHandoff(context.Background(), 1, 1)

	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation for confirming a non-accepted request, got %v", err)
	}
}

func TestMarkReturned_RejectsNonOwner(t *testing.T) {
	store := &fakeReqStore{
		getFn: func(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error) {
			return BorrowRequest{ID: id, OwnerID: 1, Status: StatusActive}, nil
		},
	}
	svc := newTestService(store, &fakeBookStore{})

	_, err := svc.MarkReturned(context.Background(), 1, 2)

	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-owner mark-returned, got %v", err)
	}
}

func TestMarkReturned_RejectsWhenNotActive(t *testing.T) {
	store := &fakeReqStore{
		getFn: func(ctx context.Context, q postgres.Querier, id int) (BorrowRequest, error) {
			return BorrowRequest{ID: id, OwnerID: 1, Status: StatusAccepted}, nil
		},
	}
	svc := newTestService(store, &fakeBookStore{})

	_, err := svc.MarkReturned(context.Background(), 1, 1)

	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation for mark-returned on a non-active request, got %v", err)
	}
}
