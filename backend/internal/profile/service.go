package profile

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"boibritto/internal/apperror"
	"boibritto/internal/books"
	"boibritto/internal/domain"
	"boibritto/internal/requests"
)

var whatsappPattern = regexp.MustCompile(`^\d{11}$`)

const (
	bookListLimit    = 10 // books shown on a profile — separate cap from the activity feed
	activityFetchCap = 5  // per-source fetch cap before merging
	activityShowCap  = 5  // final feed size after merge
)

type profileStore interface {
	UpdateProfile(ctx context.Context, userID int, name, whatsappNumber *string) error
	IsProfileComplete(ctx context.Context, userID int) (bool, error)
	GetPublicProfile(ctx context.Context, userID int) (PublicUser, error)
	GetUserByID(ctx context.Context, userID int) (*domain.User, error)
}

type booksLister interface {
	ListByOwner(ctx context.Context, ownerID, limit int) ([]books.Book, error)
}

type requestActivity interface {
	RecentSent(ctx context.Context, requesterID, limit int) ([]requests.ActivityEvent, error)
	RecentAcceptedAsOwner(ctx context.Context, ownerID, limit int) ([]requests.ActivityEvent, error)
	RecentAcceptedAsRequester(ctx context.Context, requesterID, limit int) ([]requests.ActivityEvent, error)
}

type Service struct {
	store    profileStore
	books    booksLister
	requests requestActivity
}

func NewService(store profileStore, booksLister booksLister, requestActivity requestActivity) *Service {
	return &Service{store: store, books: booksLister, requests: requestActivity}
}

// --- Profile update (unchanged from before) ---

func (s *Service) UpdateProfile(ctx context.Context, userID int, req UpdateProfileRequest) error {
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return fmt.Errorf("%w: name cannot be empty", apperror.ErrValidation)
		}
		req.Name = &trimmed
	}
	if req.WhatsAppNumber != nil {
		if !whatsappPattern.MatchString(*req.WhatsAppNumber) {
			return fmt.Errorf("%w: whatsapp_number must be exactly 11 digits", apperror.ErrValidation)
		}
	}
	if err := s.store.UpdateProfile(ctx, userID, req.Name, req.WhatsAppNumber); err != nil {
		return fmt.Errorf("updating profile: %w", err)
	}
	return nil
}

func (s *Service) IsProfileComplete(ctx context.Context, userID int) (bool, error) {
	return s.store.IsProfileComplete(ctx, userID)
}

// --- Own profile (GET /me) ---

func (s *Service) GetOwnProfile(ctx context.Context, userID int) (OwnProfileResponse, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return OwnProfileResponse{}, fmt.Errorf("getting user: %w", err)
	}

	bookList, activity, err := s.assembleBooksAndActivity(ctx, userID)
	if err != nil {
		return OwnProfileResponse{}, err
	}

	return OwnProfileResponse{
		User: UserResponse{
			ID: user.ID, Name: user.Name, Email: user.Email,
			WhatsAppNumber: user.WhatsAppNumber, LastActiveAt: user.LastActiveAt,
			CreatedAt: user.CreatedAt,
		},
		Books:          bookList,
		RecentActivity: activity,
	}, nil
}

// --- Public profile (GET /users/{id}) ---

func (s *Service) GetPublicProfile(ctx context.Context, userID int) (PublicProfileResponse, error) {
	pub, err := s.store.GetPublicProfile(ctx, userID)
	if err != nil {
		return PublicProfileResponse{}, fmt.Errorf("getting public profile: %w", err)
	}

	bookList, activity, err := s.assembleBooksAndActivity(ctx, userID)
	if err != nil {
		return PublicProfileResponse{}, err
	}

	return PublicProfileResponse{
		User: PublicUserResponse{
			ID: pub.ID, Name: pub.Name, LastActiveAt: pub.LastActiveAt, CreatedAt: pub.CreatedAt,
		},
		Books:          bookList,
		RecentActivity: activity,
	}, nil
}

// --- Shared composition logic ---

func (s *Service) assembleBooksAndActivity(ctx context.Context, userID int) ([]BookSummary, []ActivityItem, error) {
	bookRows, err := s.books.ListByOwner(ctx, userID, bookListLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("listing books: %w", err)
	}

	bookSummaries := make([]BookSummary, len(bookRows))
	items := make([]ActivityItem, 0, len(bookRows))
	for i, b := range bookRows {
		bookSummaries[i] = BookSummary{
			ID: b.ID, Title: b.Title, Author: b.Author, CoverURL: b.CoverURL, Available: b.Available,
		}
		if i < activityFetchCap {
			items = append(items, ActivityItem{
				Type:        "book_listed",
				Timestamp:   b.CreatedAt,
				Description: fmt.Sprintf("Listed \"%s\"", b.Title),
			})
		}
	}

	sent, err := s.requests.RecentSent(ctx, userID, activityFetchCap)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching sent requests: %w", err)
	}
	for _, e := range sent {
		items = append(items, ActivityItem{
			Type:        "request_sent",
			Timestamp:   e.Timestamp,
			Description: fmt.Sprintf("Requested \"%s\"", e.BookTitle),
		})
	}

	acceptedAsOwner, err := s.requests.RecentAcceptedAsOwner(ctx, userID, activityFetchCap)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching accepted-as-owner: %w", err)
	}
	for _, e := range acceptedAsOwner {
		items = append(items, ActivityItem{
			Type:        "request_accepted",
			Timestamp:   e.Timestamp,
			Description: fmt.Sprintf("Accepted a request for \"%s\"", e.BookTitle),
		})
	}

	acceptedAsRequester, err := s.requests.RecentAcceptedAsRequester(ctx, userID, activityFetchCap)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching accepted-as-requester: %w", err)
	}
	for _, e := range acceptedAsRequester {
		items = append(items, ActivityItem{
			Type:        "request_accepted",
			Timestamp:   e.Timestamp,
			Description: fmt.Sprintf("Your request for \"%s\" was accepted", e.BookTitle),
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Timestamp.After(items[j].Timestamp) })
	if len(items) > activityShowCap {
		items = items[:activityShowCap]
	}

	return bookSummaries, items, nil
}
