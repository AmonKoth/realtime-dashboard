package collector

import (
	"context"
	"realtime-dashboard/internal/metrics"
	"runtime"
	"time"
)

type RuntimeSource struct {
	interval time.Duration
}

var _ Source = (*RuntimeSource)(nil)

func NewRuntimeSource(interval time.Duration) *RuntimeSource {
	if interval <= 0 {
		interval = time.Second
	}
	return &RuntimeSource{interval: interval}
}

func (s *RuntimeSource) Name() string {
	return "runtime"
}

func (s *RuntimeSource) Run(ctx context.Context, out chan<- metrics.Event) error {
	ticker := time.NewTicker(s.interval)
	var m runtime.MemStats
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			goroutineEv := metrics.Event{
				Name:      "goroutines",
				Value:     float64(runtime.NumGoroutine()),
				Source:    s.Name(),
				Timestamp: now,
			}
			if !send(ctx, out, goroutineEv) {
				return nil
			}
			runtime.ReadMemStats(&m)
			heapEv := metrics.Event{
				Name:      "heap_mb",
				Value:     float64(m.HeapAlloc) / 1024 / 1024,
				Source:    s.Name(),
				Timestamp: now,
			}
			if !send(ctx, out, heapEv) {
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}
