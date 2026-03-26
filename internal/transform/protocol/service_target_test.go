package protocol

import "testing"

const svcURL = "http://svc.ns.svc:80"

func TestExtractLBServersVariousShapes(t *testing.T) {
	if servers, ok := extractLBServers(map[string]interface{}{}); ok || servers != nil {
		t.Fatalf("empty should be false")
	}
	if servers, ok := extractLBServers(map[string]interface{}{"x": 1}); ok || servers != nil {
		t.Fatalf("missing loadBalancer should be false")
	}
	lb := map[string]interface{}{"servers": []interface{}{map[string]interface{}{"url": "http://a"}}, "extra": 1}
	if servers, ok := extractLBServers(map[string]interface{}{"loadBalancer": lb}); ok || servers != nil {
		t.Fatalf("extra key should be false")
	}
	lb2 := map[string]interface{}{"servers": []interface{}{map[string]interface{}{"url": "http://a"}}}
	if servers, ok := extractLBServers(map[string]interface{}{"loadBalancer": lb2}); !ok || len(servers) != 1 {
		t.Fatalf("valid servers not extracted")
	}
}

func TestParseUniformServiceTargetsPositiveAndNegative(t *testing.T) {
	tests := []struct {
		name    string
		input   []interface{}
		wantOK  bool
		wantNil bool
	}{
		{
			name:    "empty input",
			input:   []interface{}{},
			wantOK:  false,
			wantNil: true,
		},
		{
			name:    "single element",
			input:   []interface{}{map[string]interface{}{"url": svcURL}},
			wantOK:  true,
			wantNil: false,
		},
		{
			name:    "uniform targets",
			input:   []interface{}{map[string]interface{}{"url": svcURL}, map[string]interface{}{"url": svcURL}},
			wantOK:  true,
			wantNil: false,
		},
		{
			name:    "mismatched urls",
			input:   []interface{}{map[string]interface{}{"url": svcURL}, map[string]interface{}{"url": "http://other.ns.svc:80"}},
			wantOK:  false,
			wantNil: true,
		},
		{
			name:    "missing url",
			input:   []interface{}{map[string]interface{}{"no": "url"}},
			wantOK:  false,
			wantNil: true,
		},
		{
			name:    "bad element type",
			input:   []interface{}{[]string{"not-a-map"}},
			wantOK:  false,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt, ok := parseUniformServiceTargets(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("parseUniformServiceTargets() ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantNil && tgt != nil {
				t.Fatalf("parseUniformServiceTargets() tgt = %v, want nil when wantNil=%v and wantOK=%v", tgt, tt.wantNil, tt.wantOK)
			}
			if !tt.wantNil && tgt == nil {
				t.Fatalf("parseUniformServiceTargets() tgt = %v, want non-nil when wantNil=%v and wantOK=%v", tgt, tt.wantNil, tt.wantOK)
			}
		})
	}
}
