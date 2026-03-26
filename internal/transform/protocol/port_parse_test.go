package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePort32Valid(t *testing.T) {
	cases := []struct {
		in   string
		want int32
	}{
		{"1", 1}, {"443", 443}, {"65535", 65535},
	}
	for _, tc := range cases {
		got, err := parsePort32(tc.in)
		require.NoError(t, err, "parsePort32(%s) unexpected error", tc.in)
		require.Equal(t, tc.want, got)
	}
}

func TestParsePort32OutOfRange(t *testing.T) {
	cases := []string{"0", "65536", "70000"}
	for _, in := range cases {
		_, err := parsePort32(in)
		require.Error(t, err, "expected error for out-of-range port %s", in)
	}
}

func TestParsePort32NonNumeric(t *testing.T) {
	cases := []string{"abc", "12x", ""}
	for _, in := range cases {
		_, err := parsePort32(in)
		require.Error(t, err, "expected error for non-numeric port %q", in)
	}
}

func TestDerivePortFromName(t *testing.T) {
	got, err := derivePortFromName("svc-8080")
	require.NoError(t, err)
	require.Equal(t, int32(8080), got)
	_, err = derivePortFromName("svc")
	require.Error(t, err)
	_, err = derivePortFromName("svc-")
	require.Error(t, err)
	_, err = derivePortFromName("svc-0")
	require.Error(t, err, "svc-0 should be rejected (zero port)")
}
