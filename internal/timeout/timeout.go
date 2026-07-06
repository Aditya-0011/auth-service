package timeout

import (
	"context"
	"time"
)

const (
	Duration = 5 * time.Second
)

func WithDeadline(c context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c, d)
}
