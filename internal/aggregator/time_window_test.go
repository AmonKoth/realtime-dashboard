package aggregator

import (
	"realtime-dashboard/internal/metrics"
	"testing"
	"time"
)

func TestWindowStats(t *testing.T) {
	base := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)

	tWindow := NewTimeWindow(time.Minute)
	tWindow.Add(base.Add(time.Second), 1)
	tWindow.Add(base.Add(time.Second*2), 2)
	tWindow.Add(base.Add(time.Second*3), 3)
	got := tWindow.Stats(base.Add(time.Second * 5))

	if got.PerSecond != 0.05 {
		t.Errorf("perSecond: got: %g, want: 0.05", got.PerSecond)
	}
	if got.Count != 3 {
		t.Errorf("count: got %d, want: 3", got.Count)
	}
	if got.Avg != 2 {
		t.Errorf("avg: got %g, want: 2", got.Avg)
	}
	if got.Last != 3 {
		t.Errorf("last: got %g, want: 3", got.Last)
	}
	if got.Max != 3 {
		t.Errorf("max: got %g, want: 3", got.Max)
	}
	if got.Min != 1 {
		t.Errorf("min: got %g, want: 1", got.Min)
	}
}

func TestWindowExpiry(t *testing.T) {
	base := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	tWindow := NewTimeWindow(time.Minute)

	tWindow.Add(base.Add(time.Second), 1)
	tWindow.Add(base.Add(time.Second*2), 2)
	tWindow.Add(base.Add(time.Second*3), 3)

	tWindow.Add(base.Add(time.Minute*2), 400)
	tWindow.Add(base.Add((time.Minute*2)+time.Second*2), 500)
	got := tWindow.Stats(base.Add((time.Minute * 2) + time.Second*6))

	if got.Count != 2 {
		t.Errorf("count: got %d, want: 2", got.Count)
	}
	if got.Last != 500 {
		t.Errorf("last: got %g, want: 500", got.Last)
	}
	if got.Max != 500 {
		t.Errorf("max: got %g, want: 500", got.Max)
	}
	if got.Min != 400 {
		t.Errorf("min: got %g, want: 400", got.Min)
	}
	if got.Avg != 450 {
		t.Errorf("avg: got %g, want: 450", got.Avg)
	}
}

func TestWindowEmpty(t *testing.T) {
	base := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	tWindow := NewTimeWindow(time.Minute)

	got := tWindow.Stats(base)
	empty := metrics.MetricStats{}
	if got != empty {
		t.Errorf("want empty got not %+v", got)
	}

	tWindow.Add(base.Add(time.Second), 1)
	tWindow.Add(base.Add(time.Second*2), 2)
	tWindow.Add(base.Add(time.Second*3), 3)
	got2 := tWindow.Stats(base.Add(time.Minute * 3))

	if got2 != empty {
		t.Errorf("want empty got %+v", got2)
	}
}

func TestWindowOutOfOrder(t *testing.T) {
	base := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	tWindow := NewTimeWindow(time.Minute)

	tWindow.Add(base.Add(time.Second*3), 3)
	tWindow.Add(base.Add(time.Second), 1)
	tWindow.Add(base.Add(time.Second*2), 2)

	got := tWindow.Stats(base.Add(time.Second * 5))
	if got.Count != 3 {
		t.Errorf("count: got %d, want: 3", got.Count)
	}

	if got.Last != 3 {
		t.Errorf("last: got %g, want: 3", got.Last)
	}

	got = tWindow.Stats(base.Add(time.Minute + time.Millisecond*1500))
	if got.Count != 2 {
		t.Errorf("count after expiry: got %d, want: 2", got.Count)
	}
	if got.Min != 2 {
		t.Errorf("min after expiry: got %g, want: 2", got.Min)
	}
}
