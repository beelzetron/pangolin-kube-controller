package controller

import "testing"

const (
	etagV1  = "\"v1\""
	etagV2  = "\"v2\""
	weakW1  = "W/\"w1\""
	etagOld = "\"old\""
)

func TestIsWeakETag(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	cases := []struct {
		et   string
		want bool
	}{
		{"W/\"abc\"", true},
		{"\"abc\"", false},
		{"", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.et, func(t *testing.T) {
			t.Parallel()
			got := ctrl.isWeakETag(tc.et)
			if got != tc.want {
				t.Fatalf("isWeakETag(%q) = %v; want %v", tc.et, got, tc.want)
			}
		})
	}
}

func TestComputeHashAndDecideChange(t *testing.T) {
	ctrl := &Controller{}
	body1 := []byte("hello")
	body2 := []byte("hello")
	body3 := []byte("world")

	h1 := ctrl.computeHash(body1)
	h2 := ctrl.computeHash(body2)
	h3 := ctrl.computeHash(body3)
	if h1 != h2 {
		t.Fatalf("hash mismatch for identical bodies")
	}
	if h1 == h3 {
		t.Fatalf("hash collision unexpected")
	}

	if !ctrl.decideChange("", "", "", body1, false) {
		t.Fatalf("expected change when no prior signatures present")
	}

	if ctrl.decideChange("", "", h1, body2, false) {
		t.Fatalf("unexpected change when bodies identical and no etag")
	}

	if ctrl.decideChange(etagV1, etagV1, h1, body2, true) {
		t.Fatalf("unexpected change when strong etag unchanged")
	}

	if ctrl.decideChange(weakW1, etagOld, h1, body2, true) {
		t.Fatalf("unexpected change when weak etag but body identical")
	}

	if ctrl.decideChange(etagV2, etagV1, h1, body2, true) {
		t.Fatalf("unexpected change when strong etag changed but body identical")
	}

	if !ctrl.decideChange(etagV2, etagV1, h1, body3, true) {
		t.Fatalf("expected change when body changed")
	}
}
