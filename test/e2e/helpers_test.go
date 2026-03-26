package e2e

import (
	"testing"
)

const defaultTag = "vX.Y.Z"

func TestTraefikVersionEnvOverride(t *testing.T) {
	const sampleTag = "v9.9.9"
	cases := []struct {
		name   string
		envVal string
		want   string
	}{
		{name: "simple override", envVal: sampleTag, want: sampleTag},
		{name: "empty string falls back", envVal: "", want: defaultTag},
		{name: "whitespace falls back", envVal: "   ", want: defaultTag},
		{name: "whitespace trimmed", envVal: " " + sampleTag + " ", want: sampleTag},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TRAEFIK_CRD_VERSION", tc.envVal)
			got := TraefikVersion(defaultTag)
			if got != tc.want {
				t.Fatalf("env override mismatch: name=%q, want=%q, got=%q", tc.name, tc.want, got)
			}
		})
	}
}
