package metrics

import (
	"testing"
	"time"
)

func TestEventValidate(t *testing.T) {

	tests := []struct {
		name    string
		ev      Event
		wantErr bool
	}{{
		name: "valid event",
		ev: Event{
			Name:      "page_view",
			Value:     1,
			Source:    "testSource",
			Timestamp: time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
		},
		wantErr: false,
	},
		{
			name: "empty name",
			ev: Event{
				Value:     1,
				Source:    "testSource",
				Timestamp: time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
		{
			name: "empty source",
			ev: Event{
				Name:      "page_view",
				Value:     1,
				Timestamp: time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
		{
			name: "zero timestamp",
			ev: Event{
				Name:   "page_view",
				Value:  1,
				Source: "testSource",
			},
			wantErr: true,
		},
		{
			name:    "empty event",
			ev:      Event{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := false
			err := tt.ev.Validate()
			if err != nil {
				result = true
			}
			if result != tt.wantErr {
				t.Errorf("got %v, want %v", result, tt.wantErr)
			}
		})
	}
}
