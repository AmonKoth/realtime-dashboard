package aggregator

import (
	"slices"
	"sort"
	"time"

	"realtime-dashboard/internal/metrics"
)

const defaultWindowSize = time.Minute

type sample struct {
	at    time.Time
	value float64
}

// TimeWindow should only have one owner so no lock will be needed
type TimeWindow struct {
	size    time.Duration
	samples []sample
}

func NewTimeWindow(size time.Duration) *TimeWindow {
	if size <= 0 {
		size = defaultWindowSize
	}
	return &TimeWindow{size: size}
}

func (w *TimeWindow) Add(at time.Time, value float64) {
	s := sample{
		at:    at,
		value: value,
	}
	if len(w.samples) == 0 || !at.Before(w.samples[len(w.samples)-1].at) {
		w.samples = append(w.samples, s)
		return
	}
	i := w.indexAfterTime(at)
	w.samples = slices.Insert(w.samples, i, s)
}

func (w *TimeWindow) evict(now time.Time) {
	cutoff := now.Add(-w.size)
	w.samples = w.samples[w.indexAfterTime(cutoff):]
}

func (w *TimeWindow) Stats(now time.Time) metrics.MetricStats {
	w.evict(now)
	if len(w.samples) == 0 {
		return metrics.MetricStats{}
	}
	count := len(w.samples)
	last := w.samples[count-1].value
	maximum, minimum := w.samples[0].value, w.samples[0].value
	sum := 0.0
	for _, s := range w.samples {
		minimum = min(minimum, s.value)
		maximum = max(maximum, s.value)
		sum += s.value
	}
	avg := sum / float64(count)
	perSecond := float64(count) / w.size.Seconds()
	return metrics.MetricStats{
		Count:     count,
		PerSecond: perSecond,
		Avg:       avg,
		Min:       minimum,
		Max:       maximum,
		Last:      last,
	}
}

func (w *TimeWindow) indexAfterTime(t time.Time) int {
	return sort.Search(len(w.samples), func(i int) bool {
		return w.samples[i].at.After(t)
	})
}
