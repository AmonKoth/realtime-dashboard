package collector

import (
	"context"
	"realtime-dashboard/internal/metrics"
	"testing"
	"time"
)

func TestSourcesStopOnCancel(t *testing.T) {
	sources := []Source{
		NewRuntimeSource(10 * time.Millisecond),
		NewWikimediaSource(),
	}

	for _, src := range sources {
		t.Run(src.Name(), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			events := make(chan metrics.Event, 100)
			go func() {
				for range events {
				}
			}()

			done := make(chan struct{})
			go func() {
				defer close(done)
				if err := src.Run(ctx, events); err != nil {
					t.Errorf("Run returned error: %v", err)
				}
			}()

			time.Sleep(100 * time.Millisecond)
			cancel()

			select {
			case <-done:
				close(events)
			case <-time.After(time.Second):
				t.Fatal("Run did not stop within 1s after cancel")
			}
		})
	}
}
