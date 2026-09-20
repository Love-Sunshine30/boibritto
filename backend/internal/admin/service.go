package admin

import (
	"context"
	"fmt"
	"strings"

	"boibritto/internal/apperror"
	"boibritto/internal/books"
)

type bookCoverSetter interface {
	SetCover(ctx context.Context, bookID int, coverURL string) (books.Book, error)
}

type Service struct {
	books bookCoverSetter
}

func NewService(books bookCoverSetter) *Service {
	return &Service{books: books}
}

func (s *Service) SetBookCover(ctx context.Context, bookID int, coverURL string) (books.Book, error) {
	coverURL = strings.TrimSpace(coverURL)
	if coverURL == "" {
		return books.Book{}, fmt.Errorf("%w: cover_url is required", apperror.ErrValidation)
	}
	updated, err := s.books.SetCover(ctx, bookID, coverURL)
	if err != nil {
		return books.Book{}, fmt.Errorf("setting cover: %w", err)
	}
	return updated, nil
}
