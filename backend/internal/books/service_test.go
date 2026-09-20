package books

import (
	"context"
	"errors"
	"testing"
	"time"

	"boibritto/internal/apperror"
)

type fakeStore struct {
	insertFn func(ctx context.Context, ownerID int, req CreateBookRequest) (Book, error)
	getFn    func(ctx context.Context, id int) (Book, error)
	updateFn func(ctx context.Context, id int, req UpdateBookRequest) (Book, error)
	deleteFn func(ctx context.Context, id int) error
	listFn   func(ctx context.Context, filter ListBooksFilter) ([]Book, error)
}

func (f *fakeStore) InsertBook(ctx context.Context, ownerID int, req CreateBookRequest) (Book, error) {
	return f.insertFn(ctx, ownerID, req)
}
func (f *fakeStore) GetBookByID(ctx context.Context, id int) (Book, error) { return f.getFn(ctx, id) }
func (f *fakeStore) UpdateBook(ctx context.Context, id int, req UpdateBookRequest) (Book, error) {
	return f.updateFn(ctx, id, req)
}
func (f *fakeStore) DeleteBook(ctx context.Context, id int) error { return f.deleteFn(ctx, id) }
func (f *fakeStore) ListBooks(ctx context.Context, filter ListBooksFilter) ([]Book, error) {
	return f.listFn(ctx, filter)
}

type fakeProfileChecker struct {
	complete bool
	err      error
}

func (f *fakeProfileChecker) IsProfileComplete(ctx context.Context, userID int) (bool, error) {
	return f.complete, f.err
}

func TestCreateBook_RejectsIncompleteProfile(t *testing.T) {
	svc := NewService(&fakeStore{}, &fakeProfileChecker{complete: false})

	_, err := svc.CreateBook(context.Background(), 1, CreateBookRequest{Title: "X", Author: "Y"})

	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateBook_RejectsEmptyTitle(t *testing.T) {
	svc := NewService(&fakeStore{}, &fakeProfileChecker{complete: true})

	_, err := svc.CreateBook(context.Background(), 1, CreateBookRequest{Title: "", Author: "Someone"})

	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateBook_TrimsWhitespaceAndSucceeds(t *testing.T) {
	var capturedTitle string
	store := &fakeStore{
		insertFn: func(ctx context.Context, ownerID int, req CreateBookRequest) (Book, error) {
			capturedTitle = req.Title
			return Book{ID: 1, Title: req.Title, OwnerID: ownerID}, nil
		},
	}
	svc := NewService(store, &fakeProfileChecker{complete: true})

	_, err := svc.CreateBook(context.Background(), 1, CreateBookRequest{Title: "  Dune  ", Author: "Herbert"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTitle != "Dune" {
		t.Errorf("expected trimmed title %q, got %q", "Dune", capturedTitle)
	}
}

func TestCreateBook_RejectsOverlongTitle(t *testing.T) {
	svc := NewService(&fakeStore{}, &fakeProfileChecker{complete: true})

	longTitle := make([]byte, 201)
	for i := range longTitle {
		longTitle[i] = 'a'
	}

	_, err := svc.CreateBook(context.Background(), 1, CreateBookRequest{Title: string(longTitle), Author: "X"})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation for overlong title, got %v", err)
	}
}

// --- Ownership check tests — this is the class of bug the original
// codebase shipped with (missing ownership checks on edit/delete). These
// tests exist specifically so that vulnerability class can't silently
// regress in this rebuild.

func TestUpdateBook_RejectsNonOwner(t *testing.T) {
	store := &fakeStore{
		getFn: func(ctx context.Context, id int) (Book, error) {
			return Book{ID: id, OwnerID: 1}, nil // owned by user 1
		},
	}
	svc := NewService(store, &fakeProfileChecker{complete: true})

	title := "New Title"
	_, err := svc.UpdateBook(context.Background(), 1, 2, UpdateBookRequest{Title: &title}) // caller is user 2

	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-owner update, got %v", err)
	}
}

func TestUpdateBook_AllowsOwner(t *testing.T) {
	updateCalled := false
	store := &fakeStore{
		getFn: func(ctx context.Context, id int) (Book, error) {
			return Book{ID: id, OwnerID: 1}, nil
		},
		updateFn: func(ctx context.Context, id int, req UpdateBookRequest) (Book, error) {
			updateCalled = true
			return Book{ID: id, OwnerID: 1, Title: *req.Title}, nil
		},
	}
	svc := NewService(store, &fakeProfileChecker{complete: true})

	title := "New Title"
	_, err := svc.UpdateBook(context.Background(), 1, 1, UpdateBookRequest{Title: &title}) // caller IS the owner

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Error("expected store.UpdateBook to be called for the actual owner")
	}
}

func TestDeleteBook_RejectsNonOwner(t *testing.T) {
	store := &fakeStore{
		getFn: func(ctx context.Context, id int) (Book, error) {
			return Book{ID: id, OwnerID: 1}, nil
		},
	}
	svc := NewService(store, &fakeProfileChecker{complete: true})

	err := svc.DeleteBook(context.Background(), 1, 2)

	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-owner delete, got %v", err)
	}
}

func TestDeleteBook_AllowsOwner(t *testing.T) {
	deleteCalled := false
	store := &fakeStore{
		getFn: func(ctx context.Context, id int) (Book, error) {
			return Book{ID: id, OwnerID: 1}, nil
		},
		deleteFn: func(ctx context.Context, id int) error {
			deleteCalled = true
			return nil
		},
	}
	svc := NewService(store, &fakeProfileChecker{complete: true})

	err := svc.DeleteBook(context.Background(), 1, 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected store.DeleteBook to be called for the actual owner")
	}
}

func TestUpdateBook_PropagatesNotFound(t *testing.T) {
	store := &fakeStore{
		getFn: func(ctx context.Context, id int) (Book, error) {
			return Book{}, apperror.ErrNotFound
		},
	}
	svc := NewService(store, &fakeProfileChecker{complete: true})

	title := "X"
	_, err := svc.UpdateBook(context.Background(), 999, 1, UpdateBookRequest{Title: &title})

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

var _ = time.Now // placeholder if unused imports trimmed differently in your actual file
