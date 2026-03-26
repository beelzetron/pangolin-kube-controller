package config

import (
	"encoding/json"
	"testing"
)

func TestGroupVersionIsComposite(t *testing.T) {
	// GroupVersion must be the concatenation of Group and Version with a slash;
	// this validates the constant definition rather than a hardcoded string.
	want := Group + "/" + Version
	if GroupVersion != want {
		t.Errorf("GroupVersion = %q, want %q (Group=%q Version=%q)", GroupVersion, want, Group, Version)
	}
}

func TestConfigJSONRoundtrip(t *testing.T) {
	raw := json.RawMessage(`{"foo":"bar"}`)
	orig := Config{
		HTTP: HTTPConfig{
			Middlewares:       map[string]json.RawMessage{"mw1": raw},
			Routers:           map[string]json.RawMessage{"r1": raw},
			Services:          map[string]json.RawMessage{"svc1": raw},
			ServersTransports: map[string]json.RawMessage{"st1": raw},
		},
	}

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if _, ok := decoded.HTTP.Middlewares["mw1"]; !ok {
		t.Error("decoded.HTTP.Middlewares missing mw1")
	}
	if _, ok := decoded.HTTP.Routers["r1"]; !ok {
		t.Error("decoded.HTTP.Routers missing r1")
	}
	if _, ok := decoded.HTTP.Services["svc1"]; !ok {
		t.Error("decoded.HTTP.Services missing svc1")
	}
	if _, ok := decoded.HTTP.ServersTransports["st1"]; !ok {
		t.Error("decoded.HTTP.ServersTransports missing st1")
	}
	if decoded.TCP != nil {
		t.Error("TCP should be nil when omitted")
	}
	if decoded.UDP != nil {
		t.Error("UDP should be nil when omitted")
	}
}

func TestConfigWithTCPAndUDP(t *testing.T) {
	raw := json.RawMessage(`{"x":1}`)
	orig := Config{
		HTTP: HTTPConfig{
			Middlewares:       map[string]json.RawMessage{},
			Routers:           map[string]json.RawMessage{},
			Services:          map[string]json.RawMessage{},
			ServersTransports: map[string]json.RawMessage{},
		},
		TCP: &TCPUDPConfig{
			Routers:           map[string]json.RawMessage{"tcpR": raw},
			Services:          map[string]json.RawMessage{"tcpS": raw},
			ServersTransports: map[string]json.RawMessage{"tcpST": raw},
		},
		UDP: &TCPUDPConfig{
			Routers:           map[string]json.RawMessage{"udpR": raw},
			Services:          map[string]json.RawMessage{},
			ServersTransports: map[string]json.RawMessage{},
		},
	}

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.TCP == nil {
		t.Fatal("decoded.TCP should not be nil")
	}
	if _, ok := decoded.TCP.Routers["tcpR"]; !ok {
		t.Error("decoded.TCP.Routers missing tcpR")
	}
	if decoded.UDP == nil {
		t.Fatal("decoded.UDP should not be nil")
	}
	if _, ok := decoded.UDP.Routers["udpR"]; !ok {
		t.Error("decoded.UDP.Routers missing udpR")
	}
}

func TestConfigEmptyJSON(t *testing.T) {
	var cfg Config
	if err := json.Unmarshal([]byte(`{}`), &cfg); err != nil {
		t.Fatalf("Unmarshal empty JSON: %v", err)
	}
	if cfg.TCP != nil || cfg.UDP != nil {
		t.Error("TCP/UDP should be nil for empty JSON")
	}
}

func TestTCPUDPConfigJSONFieldNames(t *testing.T) {
	raw := json.RawMessage(`{"k":"v"}`)
	cfg := Config{
		TCP: &TCPUDPConfig{
			Routers:           map[string]json.RawMessage{"r": raw},
			Services:          map[string]json.RawMessage{"s": raw},
			ServersTransports: map[string]json.RawMessage{"t": raw},
		},
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}
	tcpRaw, ok := m["tcp"]
	if !ok {
		t.Fatal("tcp key missing from JSON output")
	}
	tcpMap, ok := tcpRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("tcp value not a map: %T", tcpRaw)
	}
	for _, key := range []string{"routers", "services", "serversTransports"} {
		if _, has := tcpMap[key]; !has {
			t.Errorf("tcp.%s missing from JSON output", key)
		}
	}
}
