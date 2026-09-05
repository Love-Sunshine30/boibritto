package push

import (
	"context"
	"fmt"
)

// RequestNotifier adapts the generic Sender into the specific Notifier
// interface internal/requests depends on — keeps requests decoupled from
// push's Payload/Sender shape, only knowing about the one method it needs.
type RequestNotifier struct {
	sender Sender
}

func NewRequestNotifier(sender Sender) *RequestNotifier {
	return &RequestNotifier{sender: sender}
}

func (n *RequestNotifier) NotifyHandoffConfirmationNeeded(ctx context.Context, recipientUserID, requestID int) error {
	return n.sender.Send(ctx, recipientUserID, Payload{
		Title: "Confirm book handoff",
		Body:  "The other party confirmed the exchange — please confirm on your end too.",
		Data: map[string]string{
			"type":       "handoff_confirmation_needed",
			"request_id": fmt.Sprintf("%d", requestID),
		},
	})
}
