// internal/push/message_notifier.go
package push

import (
	"context"
	"fmt"
)

type MessageNotifier struct{ sender Sender }

func NewMessageNotifier(sender Sender) *MessageNotifier { return &MessageNotifier{sender: sender} }

func (n *MessageNotifier) NotifyNewMessage(ctx context.Context, recipientUserID int, senderName, preview string, requestID int) error {
	return n.sender.Send(ctx, recipientUserID, Payload{
		Title: senderName,
		Body:  preview,
		Data:  map[string]string{"type": "new_message", "request_id": fmt.Sprintf("%d", requestID)},
	})
}
