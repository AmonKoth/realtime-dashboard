package metrics

import (
	"errors"
	"fmt"
	"time"
)

var ErrInvalidEvent = errors.New("invalid event")

type Event struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}
type MetricStats struct {
	Count     int     `json:"count"`
	PerSecond float64 `json:"perSecond"`
	Avg       float64 `json:"avg"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Last      float64 `json:"last"`
}

func (e Event) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("%w: name is empty", ErrInvalidEvent)
	}
	if e.Source == "" {
		return fmt.Errorf("%w: source is empty", ErrInvalidEvent)
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("%w: timestamp is zero", ErrInvalidEvent)
	}
	return nil
}

type Snapshot struct {
	Time    time.Time              `json:"time"`
	Metrics map[string]MetricStats `json:"metrics"`
}
