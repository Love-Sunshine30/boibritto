package books

import (
	"time"
)

// CreateBookRequest is the JSON body for POST /api/v1/books.
type CreateBookRequest struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Genre       string `json:"genre"`
	Description string `json:"description"`
	CoverURL    string `json:"cover_url"`
}

// BookResponse is what the API returns — deliberately separate from the
// store's internal Book struct, so the DB schema can evolve without
// silently changing the API contract.
type BookResponse struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Genre       string    `json:"genre"`
	Description string    `json:"description"`
	CoverURL    string    `json:"cover_url"`
	Available   bool      `json:"available"`
	OwnerID     int       `json:"owner_id"`
	OwnerName   string    `json:"owner_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// UpdateBookRequest — all fields optional; only non-nil fields are updated.
type UpdateBookRequest struct {
	Title       *string `json:"title"`
	Author      *string `json:"author"`
	Genre       *string `json:"genre"`
	Description *string `json:"description"`
	CoverURL    *string `json:"cover_url"`
	Available   *bool   `json:"available"`
}

type ListBooksResponse struct {
	Books      []BookResponse `json:"books"`
	NextCursor *time.Time     `json:"next_cursor,omitempty"`
}

func toResponse(b Book) BookResponse {
	return BookResponse{
		ID: b.ID, Title: b.Title, Author: b.Author, Genre: b.Genre,
		Description: b.Description, CoverURL: b.CoverURL, Available: b.Available,
		OwnerID: b.OwnerID, OwnerName: b.OwnerName, CreatedAt: b.CreatedAt,
	}
}
