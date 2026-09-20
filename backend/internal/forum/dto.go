package forum

import "time"

type Post struct {
	ID        int       `json:"id"`
	BookID    int       `json:"book_id"`
	UserID    int       `json:"user_id"`
	UserName  string    `json:"user_name"`
	Body      string    `json:"body"`
	Edited    bool      `json:"edited"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatePostRequest struct {
	Body string `json:"body"`
}

type UpdatePostRequest struct {
	Body string `json:"body"`
}

type ListPostsResponse struct {
	Posts      []Post     `json:"posts"`
	NextCursor *time.Time `json:"next_cursor,omitempty"`
}
