package notifier

import (
	"context"
)

// Notifier 是发送通知的接口
type Notifier interface {
	Send(ctx context.Context, content string) error
}
