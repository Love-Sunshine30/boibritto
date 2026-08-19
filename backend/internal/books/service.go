package books

import (
	"context"
	"fmt"
	"strings"
	"time"

	"boibritto/internal/apperror"
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListBooks(ctx context.Context, cursor *time.Time) ([]BookResponse, error) {
	books, err := s.store.ListBooks(ctx, cursor)
	if err != nil {
		return nil, fmt.Errorf("listing books: %w", err)
	}
	resp := make([]BookResponse, len(books))
	for i, b := range books {
		resp[i] = toResponse(b)
	}
	return resp, nil
}

func (s *Service) CreateBook(ctx context.Context, ownerID int, req CreateBookRequest) (BookResponse, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Author = strings.TrimSpace(req.Author)

	if req.Title == "" || req.Author == "" {
		return BookResponse{}, fmt.Errorf("%w: title and author are required", apperror.ErrValidation)
	}
	if len(req.Title) > 200 || len(req.Description) > 2000 {
		return BookResponse{}, fmt.Errorf("%w: title or description too long", apperror.ErrValidation)
	}

	b, err := s.store.InsertBook(ctx, ownerID, req)
	if err != nil {
		return BookResponse{}, fmt.Errorf("creating book: %w", err)
	}
	return toResponse(b), nil
}

func (s *Service) UpdateBook(ctx context.Context, bookID, requesterID int, req UpdateBookRequest) (BookResponse, error) {
	existing, err := s.store.GetBookByID(ctx, bookID)
	if err != nil {
		return BookResponse{}, fmt.Errorf("looking up book: %w", err)
	}
	if existing.OwnerID != requesterID {
		return BookResponse{}, fmt.Errorf("%w: you don't own this book", apperror.ErrForbidden)
	}

	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			return BookResponse{}, fmt.Errorf("%w: title cannot be empty", apperror.ErrValidation)
		}
		req.Title = &trimmed
	}

	b, err := s.store.UpdateBook(ctx, bookID, req)
	if err != nil {
		return BookResponse{}, fmt.Errorf("updating book: %w", err)
	}
	return toResponse(b), nil
}

func (s *Service) DeleteBook(ctx context.Context, bookID, requesterID int) error {
	existing, err := s.store.GetBookByID(ctx, bookID)
	if err != nil {
		return fmt.Errorf("looking up book: %w", err)
	}
	if existing.OwnerID != requesterID {
		return fmt.Errorf("%w: you don't own this book", apperror.ErrForbidden)
	}

	if err := s.store.DeleteBook(ctx, bookID); err != nil {
		return fmt.Errorf("deleting book: %w", err)
	}
	return nil
}
