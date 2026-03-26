package reconcile

import (
	"testing"
	"time"
)

func TestNewGC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		workers     int
		gracePeriod time.Duration
		queueSize   int
		wantWorkers int
	}{
		{
			name:        "default workers when zero",
			workers:     0,
			gracePeriod: time.Second,
			queueSize:   100,
			wantWorkers: 1,
		},
		{
			name:        "default workers when negative",
			workers:     -5,
			gracePeriod: time.Second,
			queueSize:   100,
			wantWorkers: 1,
		},
		{
			name:        "default queue size when zero",
			workers:     2,
			gracePeriod: time.Second,
			queueSize:   0,
			wantWorkers: 2,
		},
		{
			name:        "default queue size when negative",
			workers:     2,
			gracePeriod: time.Second,
			queueSize:   -10,
			wantWorkers: 2,
		},
		{
			name:        "positive values unchanged",
			workers:     4,
			gracePeriod: 5 * time.Second,
			queueSize:   500,
			wantWorkers: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gc := NewGC(tt.workers, tt.gracePeriod, tt.queueSize)
			if got := gc.Workers(); got != tt.wantWorkers {
				t.Errorf("NewGC().Workers() = %v, want %v", got, tt.wantWorkers)
			}
			if tt.gracePeriod != time.Duration(0) && gc.gracePeriod != tt.gracePeriod {
				t.Errorf("NewGC().gracePeriod = %v, want %v", gc.gracePeriod, tt.gracePeriod)
			}
		})
	}
}

func TestGCWorkers(t *testing.T) {
	t.Parallel()

	gc := NewGC(3, time.Second, 100)
	if got := gc.Workers(); got != 3 {
		t.Errorf("Workers() = %v, want 3", got)
	}
}
