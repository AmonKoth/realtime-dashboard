package collector

import (
	"context"
	"realtime-dashboard/internal/metrics"
)

type Source interface {
	Name() string
	Run(ctx context.Context, out chan<- metrics.Event) error
}

func send(ctx context.Context, out chan<- metrics.Event, ev metrics.Event) bool {
	select {
	case out <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}
