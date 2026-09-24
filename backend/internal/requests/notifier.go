package requests

import "context"

type Notifier interface {
	NotifyHandoffConfirmationNeeded(ctx context.Context, recipientUserID, requestID int) error
}

type NoopNotifier struct{}

func (NoopNotifier) NotifyHandoffConfirmationNeeded(ctx context.Context, recipientUserID, requestID int) error {
	return nil
}
