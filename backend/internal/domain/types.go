package domain

import (
	"database/sql"
	"time"

	"github.com/alexedwards/scs/v2"
)

// sessiong manages
var sessionManager *scs.SessionManager

type App struct {
	db *sql.DB
}

type User struct {
	ID             int       `json:"id"`
	FirebaseUID    string    `json:"-"` // internal only — never expose to clients
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	WhatsAppNumber *string   `json:"whatsapp_number"`
	IsAdmin        bool      `json:"-"` // internal only — clients don't need to see this on arbitrary user objects
	CreatedAt      time.Time `json:"created_at"`
}

type Book struct {
	ID               int
	Title            string
	Author           string
	Genre            string
	Description      string
	CoverURL         string
	ISBN             string
	OwnerID          int
	OwnerName        string
	OwnerLentCount   int
	OwnerReliability string
	Available        bool
	CreatedAt        time.Time
}

type Session struct {
	UserID       int
	SessionToken string
	Flash        string
}

type BorrowRequest struct {
	ID            int
	BookID        int
	BookTitle     string
	RequesterName string
	Message       string
	Status        string
	OwnerWpNumber *string
	OwnerName     string
	CreatedAt     time.Time
}

type pageData struct {
	User                *User
	Books               []Book
	Book                *Book
	IncomingRequests    []BorrowRequest
	SentRequests        []BorrowRequest
	Request             BorrowRequest
	Flash               bool
	FlashMessage        string
	FlashType           string
	BookID              int
	BookOwner           string
	HasMore             bool
	NextCursor          string
	BooksRequestedCount int    // count of requests this user made
	BooksLentCount      int    // count of completed loans as owner
	ReliabilityText     string // e.g. "Usually returns within 9 days"
	Activity            []ActivityItem
	Threads             []ThreadPreview
	UnreadCount         int
	OtherUserName       string
	RequestStatus       string
	RequestID           int
	Messages            []MessageView
	BookTitle           string
	LastMessageID       int
	Query               string
	Genres              []string
	SelectedGenre       string
	Error               string
}

type ActivityItem struct {
	Text    string // "Lent a book", "Listed a book" — server formats this
	TimeAgo string // "4 days ago" — format server-side, don't push raw timestamps to templates
}

type InboxData struct {
	User        *User
	Flash       bool
	UnreadCount int
	Threads     []ThreadPreview
}

type ThreadPreview struct {
	RequestID     int
	OtherUserName string
	BookTitle     string
	LastMessage   string
	LastMessageAt string // pre-formatted, e.g. "2h ago"
	Unread        bool
}

type ThreadData struct {
	User          *User
	Flash         bool
	RequestID     int
	OtherUserName string
	BookTitle     string
	RequestStatus string // "pending" / "accepted" / "rejected"
	LastMessageID int    // used by JS to know where to start polling from
	Messages      []MessageView
}

type MessageView struct {
	ID        int    `json:"id"`
	Body      string `json:"body"`
	TimeLabel string `json:"timeLabel"`
	IsMine    bool   `json:"isMine"`
}

type ShelfPageData struct {
	User             *User
	Flash            bool   // nav pending-dot — unrelated to the banner below
	FlashMessage     string // NEW — the on-page banner text, renamed to avoid the collision
	FlashType        string
	UnreadCount      int
	SentRequests     []SentRequest
	IncomingRequests []IncomingRequest // pending only — template filters, but cleaner to filter in the query
	ArrangedIncoming []IncomingRequest // NEW — accepted incoming requests
	Books            []Book
}

type SentRequest struct {
	ID        int
	BookTitle string
	OwnerName string
	Message   string
	Status    string // "pending" / "accepted" / "rejected"
}

type IncomingRequest struct {
	ID            int
	BookTitle     string
	RequesterName string
	Message       string
	Status        string
}

// CoverSearchResult is the simplified shape sent to the browser —
// only what the "list a book" form actually needs to auto-fill.
type CoverSearchResult struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	ISBN     string `json:"isbn"`
	CoverURL string `json:"coverUrl"`
}
