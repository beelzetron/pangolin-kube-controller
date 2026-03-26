package routing

import (
	"encoding/json"
	"strings"
	"testing"
)

const testRulePath = "Path(`/foo`)"

func TestParseEntryPointsFromInterface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		vals []interface{}
		want []string
	}{
		{
			name: "normal strings",
			vals: []interface{}{"web", "admin"},
			want: []string{"web", "admin"},
		},
		{
			name: "with whitespace trimmed",
			vals: []interface{}{"  web  ", "  admin  "},
			want: []string{"web", "admin"},
		},
		{
			name: "empty and whitespace filtered",
			vals: []interface{}{"web", "", "  "},
			want: []string{"web"},
		},
		{
			name: "non-string types ignored",
			vals: []interface{}{"web", 123, true, "admin"},
			want: []string{"web", "admin"},
		},
		{
			name: "empty slice",
			vals: []interface{}{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseEntryPointsFromInterface(tt.vals)
			if len(got) != len(tt.want) {
				t.Errorf("parseEntryPointsFromInterface() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseEntryPointsFromInterface()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestParseEntryPointsFromStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		vals []string
		want []string
	}{
		{
			name: "normal strings",
			vals: []string{"web", "admin"},
			want: []string{"web", "admin"},
		},
		{
			name: "with whitespace trimmed",
			vals: []string{"  web  ", "  admin  "},
			want: []string{"web", "admin"},
		},
		{
			name: "empty and whitespace filtered",
			vals: []string{"web", "", "  "},
			want: []string{"web"},
		},
		{
			name: "empty slice",
			vals: []string{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseEntryPointsFromStrings(tt.vals)
			if len(got) != len(tt.want) {
				t.Errorf("parseEntryPointsFromStrings() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseEntryPointsFromStrings()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestParseMiddlewares(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		r    map[string]interface{}
		want []string
	}{
		{
			name: "nil router",
			r:    nil,
			want: nil,
		},
		{
			name: "no middlewares",
			r:    map[string]interface{}{"rule": testRulePath},
			want: nil,
		},
		{
			name: "with middlewares",
			r: map[string]interface{}{
				"middlewares": []interface{}{"mw1", "mw2"},
			},
			want: []string{"mw1", "mw2"},
		},
		{
			name: "non-string middleware ignored",
			r: map[string]interface{}{
				"middlewares": []interface{}{"mw1", 123},
			},
			want: []string{"mw1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseMiddlewares(tt.r)
			if len(got) != len(tt.want) {
				t.Errorf("parseMiddlewares() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseMiddlewares()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestBuildRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		rule         string
		serviceName  string
		mws          []string
		priorityVal  float64
		wantPriority bool
	}{
		{
			name:         "basic route",
			rule:         testRulePath,
			serviceName:  "svc1",
			mws:          nil,
			priorityVal:  0,
			wantPriority: false,
		},
		{
			name:         "route with middlewares",
			rule:         testRulePath,
			serviceName:  "svc1",
			mws:          []string{"mw1", "mw2"},
			priorityVal:  0,
			wantPriority: false,
		},
		{
			name:         "route with priority",
			rule:         testRulePath,
			serviceName:  "svc1",
			mws:          nil,
			priorityVal:  100,
			wantPriority: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildRoute(tt.rule, tt.serviceName, tt.mws, tt.priorityVal)
			assertBuildRouteCore(t, got, tt.rule)
			assertBuildRouteMiddlewares(t, got, tt.mws)
			assertBuildRoutePriority(t, got, tt.priorityVal, tt.wantPriority)
		})
	}
}

func TestTransformRouterToIngressRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     json.RawMessage
		cfg     RouterConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "invalid json",
			raw:     json.RawMessage(`{invalid`),
			wantErr: true,
			errMsg:  "unmarshal",
		},
		{
			name:    "missing rule and service",
			raw:     json.RawMessage(`{}`),
			cfg:     RouterConfig{Namespace: "ns1"},
			wantErr: true,
			errMsg:  "missing rule and service",
		},
		{
			name:    "missing rule only",
			raw:     json.RawMessage(`{"service":"svc1"}`),
			cfg:     RouterConfig{Namespace: "ns1"},
			wantErr: true,
			errMsg:  "missing rule",
		},
		{
			name:    "missing service only",
			raw:     json.RawMessage(`{"rule":"Path(/foo)"}`),
			cfg:     RouterConfig{Namespace: "ns1"},
			wantErr: true,
			errMsg:  "missing service",
		},
		{
			name:    "valid router minimal",
			raw:     json.RawMessage(`{"rule":"Path(/foo)","service":"svc1"}`),
			cfg:     RouterConfig{Namespace: "ns1"},
			wantErr: false,
		},
		{
			name: "valid router with all fields",
			raw:  json.RawMessage(`{"rule":"Path(/foo)","service":"svc1","entryPoints":["web"],"middlewares":["mw1"],"priority":100}`),
			cfg: RouterConfig{
				Namespace:         "ns1",
				ManagedLabelKey:   "app",
				ManagedLabelValue: "pangolin",
				IngressClass:      "traefik",
			},
			wantErr: false,
		},
		{
			name:    "router with tls",
			raw:     json.RawMessage(`{"rule":"Path(/foo)","service":"svc1","tls":{}}`),
			cfg:     RouterConfig{Namespace: "ns1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := TransformRouterToIngressRoute("test-router", tt.raw, tt.cfg)
			if tt.wantErr {
				assertTransformRouterError(t, err, tt.errMsg)
				return
			}
			assertTransformRouterSuccess(t, got, err)
		})
	}
}

func assertBuildRouteCore(t *testing.T, got map[string]interface{}, wantRule string) {
	t.Helper()
	if got["match"] != wantRule {
		t.Errorf("buildRoute() match = %q, want %q", got["match"], wantRule)
	}
	if got["kind"] != "Rule" {
		t.Errorf("buildRoute() kind = %q, want %q", got["kind"], "Rule")
	}
	services, ok := got["services"].([]interface{})
	if !ok || len(services) != 1 {
		t.Errorf("buildRoute() services missing or wrong type")
	}
}

func assertBuildRouteMiddlewares(t *testing.T, got map[string]interface{}, expected []string) {
	t.Helper()
	if expected == nil {
		if _, ok := got["middlewares"]; ok {
			t.Errorf("buildRoute() should not have middlewares key when mws is nil")
		}
		return
	}
	mwRefs, ok := got["middlewares"].([]interface{})
	if !ok || len(mwRefs) != len(expected) {
		t.Errorf("buildRoute() middlewares len = %d, want %d", len(mwRefs), len(expected))
	}
}

func assertBuildRoutePriority(t *testing.T, got map[string]interface{}, priorityVal float64, wantPriority bool) {
	t.Helper()
	if !wantPriority {
		return
	}
	pri, ok := got["priority"]
	if !ok {
		t.Errorf("buildRoute() missing priority")
		return
	}
	if pri != int(priorityVal) {
		t.Errorf("buildRoute() priority = %v, want %v", pri, int(priorityVal))
	}
}

func assertTransformRouterError(t *testing.T, err error, errMsg string) {
	t.Helper()
	if err == nil {
		t.Errorf("TransformRouterToIngressRoute() expected error containing %q, got nil", errMsg)
		return
	}
	if errMsg != "" && !strings.Contains(err.Error(), errMsg) {
		t.Errorf("TransformRouterToIngressRoute() error = %v, want containing %q", err, errMsg)
	}
}

func assertTransformRouterSuccess(t *testing.T, got map[string]interface{}, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("TransformRouterToIngressRoute() unexpected error: %v", err)
		return
	}
	if got["kind"] != KindIngressRoute {
		t.Errorf("TransformRouterToIngressRoute() kind = %q, want %q", got["kind"], KindIngressRoute)
	}
	if got["apiVersion"] != TraefikAPIVersion {
		t.Errorf("TransformRouterToIngressRoute() apiVersion = %q, want %q", got["apiVersion"], TraefikAPIVersion)
	}
}

func TestAnnotateRouterEntryPointsIfPresent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		u         map[string]interface{}
		meta      map[string]interface{}
		wantKey   string
		wantValue string
	}{
		{
			name:    "nil u returns",
			u:       nil,
			meta:    nil,
			wantKey: "",
		},
		{
			name:    "nil meta returns",
			u:       map[string]interface{}{},
			meta:    nil,
			wantKey: "",
		},
		{
			name:    "no spec returns",
			u:       map[string]interface{}{},
			meta:    map[string]interface{}{},
			wantKey: "",
		},
		{
			name:    "no entryPoints returns",
			u:       map[string]interface{}{"spec": map[string]interface{}{}},
			meta:    map[string]interface{}{},
			wantKey: "",
		},
		{
			name:      "adds annotation",
			u:         map[string]interface{}{"spec": map[string]interface{}{"entryPoints": []interface{}{"web", "admin"}}},
			meta:      map[string]interface{}{},
			wantKey:   RouterEntryPointsAnnotation,
			wantValue: "web,admin",
		},
		{
			name:      "existing annotations preserved",
			u:         map[string]interface{}{"spec": map[string]interface{}{"entryPoints": []interface{}{"web"}}},
			meta:      map[string]interface{}{"annotations": map[string]interface{}{"foo": "bar"}},
			wantKey:   RouterEntryPointsAnnotation,
			wantValue: "web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			AnnotateRouterEntryPointsIfPresent(tt.u, tt.meta)
			if tt.wantKey == "" {
				return
			}
			anns, ok := tt.meta["annotations"].(map[string]interface{})
			if !ok {
				t.Errorf("AnnotateRouterEntryPointsIfPresent() annotations not set")
				return
			}
			if v, ok := anns[tt.wantKey]; !ok {
				t.Errorf("AnnotateRouterEntryPointsIfPresent() key %q not found", tt.wantKey)
			} else if v != tt.wantValue {
				t.Errorf("AnnotateRouterEntryPointsIfPresent() value = %q, want %q", v, tt.wantValue)
			}
		})
	}
}
