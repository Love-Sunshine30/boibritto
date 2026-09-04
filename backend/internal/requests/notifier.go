package requests

import "context"

// Notifier is the narrow interface requests depends on for notifying the
// other party of handoff-confirmation events. Implemented for real once
// internal/push exists; a no-op implementation lets this package be built
// and tested now without that dependency existing yet.
type Notifier interface {
	NotifyHandoffConfirmationNeeded(ctx context.Context, recipientUserID, requestID int) error
}

type NoopNotifier struct{}

func (NoopNotifier) NotifyHandoffConfirmationNeeded(ctx context.Context, recipientUserID, requestID int) error {
	return nil
}
