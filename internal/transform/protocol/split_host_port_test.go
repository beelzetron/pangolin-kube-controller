package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitHostPortFlexible(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		input     string
		host      string
		port      int32
		expectErr bool
	}{
		{"host:port", "example.com:8080", "example.com", 8080, false},
		{"url with port", "http://ex.org:9090", "ex.org", 9090, false},
		{"url missing port", "http://ex.org", "", 0, true},
		{"raw missing colon", "notcolonstring", "", 0, true},
		{"ipv6 bracket", "[::1]:8080", "::1", 8080, false},
		{"port zero", "example.org:0", "", 0, true},
		{"port max", "example.net:65535", "example.net", 65535, false},
		{"port out of range", "example.io:65536", "", 0, true},
		{"empty input", "", "", 0, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotHost, gotPort, err := splitHostPortFlexible(tc.input)
			if tc.expectErr {
				require.Error(t, err, "expected error for %s", tc.input)
				return
			}
			require.NoError(t, err, "unexpected error for %s", tc.input)
			require.Equal(t, tc.host, gotHost)
			require.Equal(t, tc.port, gotPort)
		})
	}
}
