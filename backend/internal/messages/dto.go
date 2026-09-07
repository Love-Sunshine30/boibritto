package messages

import "time"

type ThreadSummary struct {
	RequestID          int       `json:"request_id"`
	BookTitle          string    `json:"book_title"`
	OtherParticipant   string    `json:"other_participant_name"`
	LastMessagePreview string    `json:"last_message_preview"`
	LastMessageAt      time.Time `json:"last_message_at"`
}

type MessageResponse struct {
	SenderID  int       `json:"sender_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
}
