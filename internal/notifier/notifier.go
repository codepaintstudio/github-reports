package notifier

import (
	"context"
)

// Notifier is the interface for sending notifications
type Notifier interface {
	Send(ctx context.Context, content string) error
}
