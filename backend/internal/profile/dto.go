package profile

import (
	"boibritto/internal/domain"
	"time"
)

type UpdateProfileRequest struct {
	Name           *string `json:"name"`
	WhatsAppNumber *string `json:"whatsapp_number"`
}

type UserResponse struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	WhatsAppNumber *string    `json:"whatsapp_number"`
	LastActiveAt   *time.Time `json:"last_active_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type PublicUserResponse struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	LastActiveAt *time.Time `json:"last_active_at"`
	CreatedAt    time.Time  `json:"created_at"`
	// Deliberately excluded: Email, WhatsAppNumber — these are only
	// exposed via /me (your own profile)
}

type PublicUser struct {
	ID           int
	Name         string
	LastActiveAt *time.Time
	CreatedAt    time.Time
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID: u.ID, Name: u.Name, Email: u.Email,
		WhatsAppNumber: u.WhatsAppNumber, LastActiveAt: u.LastActiveAt,
		CreatedAt: u.CreatedAt,
	}
}

type BookSummary struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	CoverURL  string `json:"cover_url"`
	Available bool   `json:"available"`
}

type ActivityItem struct {
	Type        string    `json:"type"` // "book_listed" | "request_sent" | "request_accepted"
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

type OwnProfileResponse struct {
	User           UserResponse   `json:"user"`
	Books          []BookSummary  `json:"books"`
	RecentActivity []ActivityItem `json:"recent_activity"`
}

type PublicProfileResponse struct {
	User           PublicUserResponse `json:"user"`
	Books          []BookSummary      `json:"books"`
	RecentActivity []ActivityItem     `json:"recent_activity"`
}
