package requests

import "time"

type Status string

const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
	StatusActive   Status = "active"
	StatusRejected Status = "rejected"
	StatusReturned Status = "returned"
)

type BorrowRequestResponse struct {
	ID                int       `json:"id"`
	BookID            int       `json:"book_id"`
	BookTitle         string    `json:"book_title"`
	RequesterID       int       `json:"requester_id"`
	RequesterName     string    `json:"requester_name"`
	Message           string    `json:"message"`
	OwnerID           int       `json:"owner_id"`
	Status            Status    `json:"status"`
	OwnerConfirmed    bool      `json:"owner_confirmed"`
	BorrowerConfirmed bool      `json:"borrower_confirmed"`
	CreatedAt         time.Time `json:"created_at"`
}

type CreateRequestBody struct {
	Message string `json:"message"`
}

type UpdateStatusRequest struct {
	Status Status `json:"status"` // "accepted" or "rejected" only
}

func toResponse(r BorrowRequest) BorrowRequestResponse {
	return BorrowRequestResponse{
		ID: r.ID, BookID: r.BookID, BookTitle: r.BookTitle,
		RequesterID: r.RequesterID, RequesterName: r.RequesterName,
		OwnerID: r.OwnerID, Status: r.Status, Message: r.Message,
		OwnerConfirmed: r.OwnerConfirmed, BorrowerConfirmed: r.BorrowerConfirmed,
		CreatedAt: r.CreatedAt,
	}
}

func toResponses(reqs []BorrowRequest) []BorrowRequestResponse {
	resp := make([]BorrowRequestResponse, len(reqs))
	for i, r := range reqs {
		resp[i] = toResponse(r)
	}
	return resp
}
