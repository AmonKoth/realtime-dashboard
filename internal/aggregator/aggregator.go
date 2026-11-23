package aggregator

import (
	"context"
	"realtime-dashboard/internal/metrics"
	"sync"
	"time"
)

type Aggregator struct {
	in             <-chan metrics.Event
	tick           time.Duration
	timeWindowSize time.Duration
	timeWindows    map[string]*TimeWindow

	publish func(metrics.Snapshot)
	mutex   sync.RWMutex
	latest  metrics.Snapshot
}

func NewAggregator(in <-chan metrics.Event, tick, timeWindowSize time.Duration, publish func(metrics.Snapshot)) *Aggregator {
	if tick <= 0 {
		tick = time.Second
	}
	if publish == nil {
		publish = func(snapshot metrics.Snapshot) {}
	}
	return &Aggregator{
		in:             in,
		tick:           tick,
		timeWindowSize: timeWindowSize,
		timeWindows:    make(map[string]*TimeWindow),
		publish:        publish,
	}
}

func (a *Aggregator) Run(ctx context.Context) {
	ticker := time.NewTicker(a.tick)
	defer ticker.Stop()
	for {
		select {
		case ev, ok := <-a.in:
			if !ok {
				return
			}
			timeWindow, ok := a.timeWindows[ev.Name]
			if !ok {
				timeWindow = NewTimeWindow(a.timeWindowSize)
				a.timeWindows[ev.Name] = timeWindow
			}
			timeWindow.Add(ev.Timestamp, ev.Value)
		case now := <-ticker.C:
			snapshot := a.snapshot(now)
			a.mutex.Lock()
			a.latest = snapshot
			a.mutex.Unlock()
			a.publish(snapshot)
		case <-ctx.Done():
			return
		}
	}
}

func (a *Aggregator) snapshot(now time.Time) metrics.Snapshot {
	results := make(map[string]metrics.MetricStats, len(a.timeWindows))
	for name, w := range a.timeWindows {
		stat := w.Stats(now)
		if stat.Count == 0 {
			delete(a.timeWindows, name)
			continue
		}
		results[name] = stat
	}
	return metrics.Snapshot{
		Time:    now,
		Metrics: results,
	}
}

func (a *Aggregator) Latest() metrics.Snapshot {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.latest
}
