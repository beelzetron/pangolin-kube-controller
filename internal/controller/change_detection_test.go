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

func TestDecideChangeWithETagFullMatrix(t *testing.T) {
	ctrl := &Controller{}

	tests := []struct {
		name             string
		etag             string
		lastETag         string
		lastHash         string
		body             []byte
		lastETagIsHeader bool
		wantChange       bool
		wantReason       string
	}{
		// Strong ETag cases
		{
			name:             "strong etag matches, lastETagIsHeader=true",
			etag:             `"v1"`,
			lastETag:         `"v1"`,
			lastHash:         ctrl.computeHash([]byte("same")),
			body:             []byte("same"),
			lastETagIsHeader: true,
			wantChange:       false,
			wantReason:       "strong etag match with header tracking",
		},
		{
			name:             "strong etag matches, lastETagIsHeader=false",
			etag:             `"v1"`,
			lastETag:         `"v1"`,
			lastHash:         ctrl.computeHash([]byte("same")),
			body:             []byte("same"),
			lastETagIsHeader: false,
			wantChange:       false,
			wantReason:       "lastETagIsHeader=false skips strong etag check; falls through to hash comparison",
		},
		{
			name:             "strong etag differs, hash same, lastETagIsHeader=true",
			etag:             `"v2"`,
			lastETag:         `"v1"`,
			lastHash:         ctrl.computeHash([]byte("same")),
			body:             []byte("same"),
			lastETagIsHeader: true,
			wantChange:       false,
			wantReason:       "strong etag changed but body hash unchanged",
		},
		{
			name:             "strong etag differs, hash differs",
			etag:             `"v2"`,
			lastETag:         `"v1"`,
			lastHash:         ctrl.computeHash([]byte("old")),
			body:             []byte("new"),
			lastETagIsHeader: true,
			wantChange:       true,
			wantReason:       "strong etag changed and body hash changed",
		},
		{
			name:             "strong etag differs, lastETagIsHeader=false, hash same",
			etag:             `"v2"`,
			lastETag:         `"v1"`,
			lastHash:         ctrl.computeHash([]byte("same")),
			body:             []byte("same"),
			lastETagIsHeader: false,
			wantChange:       false,
			wantReason:       "strong etag check skipped due to lastETagIsHeader=false; falls through to hash comparison",
		},
		// Weak ETag cases
		{
			name:             "weak etag, hash same",
			etag:             `W/"w1"`,
			lastETag:         `W/"old"`,
			lastHash:         ctrl.computeHash([]byte("same")),
			body:             []byte("same"),
			lastETagIsHeader: true,
			wantChange:       false,
			wantReason:       "weak etag but body hash unchanged",
		},
		{
			name:             "weak etag, hash differs",
			etag:             `W/"w1"`,
			lastETag:         `W/"old"`,
			lastHash:         ctrl.computeHash([]byte("old")),
			body:             []byte("new"),
			lastETagIsHeader: true,
			wantChange:       true,
			wantReason:       "weak etag and body hash changed",
		},
		{
			name:             "weak etag, no prior hash",
			etag:             `W/"w1"`,
			lastETag:         "",
			lastHash:         "",
			body:             []byte("new"),
			lastETagIsHeader: false,
			wantChange:       true,
			wantReason:       "weak etag but no prior hash to compare",
		},
		// No ETag present (empty string)
		{
			name:             "no etag, hash same",
			etag:             "",
			lastETag:         "",
			lastHash:         ctrl.computeHash([]byte("same")),
			body:             []byte("same"),
			lastETagIsHeader: false,
			wantChange:       false,
			wantReason:       "no etag but hash unchanged",
		},
		{
			name:             "no etag, hash differs",
			etag:             "",
			lastETag:         "",
			lastHash:         ctrl.computeHash([]byte("old")),
			body:             []byte("new"),
			lastETagIsHeader: false,
			wantChange:       true,
			wantReason:       "no etag and hash changed",
		},
		{
			name:             "no etag, no prior hash",
			etag:             "",
			lastETag:         "",
			lastHash:         "",
			body:             []byte("new"),
			lastETagIsHeader: false,
			wantChange:       true,
			wantReason:       "first run with no etag or hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ctrl.decideChange(tt.etag, tt.lastETag, tt.lastHash, tt.body, tt.lastETagIsHeader)
			if got != tt.wantChange {
				t.Errorf("decideChange(%q, %q, lastHash, body, %v) = %v; want %v (%s)",
					tt.etag, tt.lastETag, tt.lastETagIsHeader, got, tt.wantChange, tt.wantReason)
			}
		})
	}
}
