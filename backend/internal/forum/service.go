package forum

import (
	"context"
	"fmt"
	"strings"
	"time"

	"boibritto/internal/apperror"
	"boibritto/internal/books"
)

type forumStore interface {
	ListByBook(ctx context.Context, bookID int, cursor *time.Time) ([]postRow, error)
	GetByID(ctx context.Context, id int) (postRow, error)
	Insert(ctx context.Context, bookID, userID int, body string) (postRow, error)
	Update(ctx context.Context, id int, body string) (postRow, error)
	Delete(ctx context.Context, id int) error
}

type bookExists interface {
	GetBookByID(ctx context.Context, id int) (books.Book, error)
}

type profileChecker interface {
	IsProfileComplete(ctx context.Context, userID int) (bool, error)
}

type Service struct {
	store   forumStore
	books   bookExists
	profile profileChecker
}

func NewService(store forumStore, books bookExists, profile profileChecker) *Service {
	return &Service{store: store, books: books, profile: profile}
}

func (s *Service) ListPosts(ctx context.Context, bookID int, cursor *time.Time) ([]Post, *time.Time, error) {
	rows, err := s.store.ListByBook(ctx, bookID, cursor)
	if err != nil {
		return nil, nil, fmt.Errorf("listing forum posts: %w", err)
	}
	posts := make([]Post, len(rows))
	for i, r := range rows {
		posts[i] = toPost(r)
	}
	var next *time.Time
	if len(rows) == pageSize {
		next = &rows[len(rows)-1].CreatedAt
	}
	return posts, next, nil
}

func (s *Service) CreatePost(ctx context.Context, bookID, userID int, body string) (Post, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Post{}, fmt.Errorf("%w: post cannot be empty", apperror.ErrValidation)
	}
	if len(body) > 3000 {
		return Post{}, fmt.Errorf("%w: post is too long", apperror.ErrValidation)
	}

	complete, err := s.profile.IsProfileComplete(ctx, userID)
	if err != nil {
		return Post{}, fmt.Errorf("checking profile: %w", err)
	}
	if !complete {
		return Post{}, fmt.Errorf("%w: complete your profile before posting", apperror.ErrForbidden)
	}

	if _, err := s.books.GetBookByID(ctx, bookID); err != nil {
		return Post{}, fmt.Errorf("looking up book: %w", err)
	}

	row, err := s.store.Insert(ctx, bookID, userID, body)
	if err != nil {
		return Post{}, fmt.Errorf("creating post: %w", err)
	}
	return toPost(row), nil
}

func (s *Service) UpdatePost(ctx context.Context, postID, callerID int, body string) (Post, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Post{}, fmt.Errorf("%w: post cannot be empty", apperror.ErrValidation)
	}

	existing, err := s.store.GetByID(ctx, postID)
	if err != nil {
		return Post{}, fmt.Errorf("looking up post: %w", err)
	}
	if existing.UserID != callerID {
		return Post{}, fmt.Errorf("%w: you can only edit your own posts", apperror.ErrForbidden)
	}

	row, err := s.store.Update(ctx, postID, body)
	if err != nil {
		return Post{}, fmt.Errorf("updating post: %w", err)
	}
	return toPost(row), nil
}

func (s *Service) DeletePost(ctx context.Context, postID, callerID int) error {
	existing, err := s.store.GetByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("looking up post: %w", err)
	}
	if existing.UserID != callerID {
		return fmt.Errorf("%w: you can only delete your own posts", apperror.ErrForbidden)
	}
	return s.store.Delete(ctx, postID)
}

func toPost(r postRow) Post {
	return Post{
		ID: r.ID, BookID: r.BookID, UserID: r.UserID, UserName: r.UserName,
		Body: r.Body, Edited: r.UpdatedAt.After(r.CreatedAt), CreatedAt: r.CreatedAt,
	}
}
