package aggregator

import (
	"context"
	"realtime-dashboard/internal/metrics"
	"sync"
	"testing"
	"time"
)

const (
	tick           = time.Millisecond * 10
	timeWindowSize = time.Hour
)

func TestConcurrentIngest(t *testing.T) {
	snapshots := make(chan metrics.Snapshot, 10)
	events := make(chan metrics.Event)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	publish := func(s metrics.Snapshot) {
		select {
		case snapshots <- s:
		default:
		}
	}
	aggregator := NewAggregator(events, tick, timeWindowSize, publish)
	wait := sync.WaitGroup{}
	go aggregator.Run(ctx)

	for range 100 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range 100 {
				events <- metrics.Event{
					Name:      "test",
					Value:     1,
					Source:    "test source",
					Timestamp: time.Now(),
				}
			}
		}()
	}
	wait.Wait()
	timeout := time.After(time.Second * 2)
	for {
		select {
		case s := <-snapshots:
			if s.Metrics["test"].Count == 10000 {
				return
			}
		case <-timeout:
			t.Fatalf("test failed without reaching 10 000")
		}
	}
}
