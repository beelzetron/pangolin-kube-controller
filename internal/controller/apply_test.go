package controller

import (
	"encoding/json"
	"testing"
)

func TestBuildDesiredSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		objects map[string]json.RawMessage
		wantLen int
	}{
		{
			name:    "empty map",
			objects: map[string]json.RawMessage{},
			wantLen: 0,
		},
		{
			name:    "nil map",
			objects: nil,
			wantLen: 0,
		},
		{
			name: "single item",
			objects: map[string]json.RawMessage{
				"svc1": json.RawMessage("{}"),
			},
			wantLen: 1,
		},
		{
			name: "multiple items",
			objects: map[string]json.RawMessage{
				"svc1": json.RawMessage("{}"),
				"svc2": json.RawMessage(`{"foo":"bar"}`),
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildDesiredSet(tt.objects)
			if len(got) != tt.wantLen {
				t.Errorf("buildDesiredSet() len = %d, want %d", len(got), tt.wantLen)
			}
			for name := range tt.objects {
				if _, ok := got[name]; !ok {
					t.Errorf("buildDesiredSet() missing key %q", name)
				}
			}
		})
	}
}
