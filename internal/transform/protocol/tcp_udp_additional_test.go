package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	traefikconfig "pangolin-kube-controller/internal/transform/config"
)

// TestBuildTCPRoutesUnmarshalError ensures we surface JSON errors from router specs.
func TestBuildTCPRoutesUnmarshalError(t *testing.T) {
	nameMap := map[string]string{"svc": "svc-8080"}
	svcInfo := map[string]tcpSvcInfo{"svc": {port: 8080, hosts: []string{"h"}, transport: "x"}}
	routers := map[string]json.RawMessage{"bad": json.RawMessage("not-json")}
	_, err := buildTCPRoutes("ns", routers, nameMap, svcInfo)
	require.Error(t, err)
}

// TestBuildTCPRoutesNoInjectionWhenTransportMissing ensures no serversTransport injection
// when the referenced dynamic service has no transport configured.
func TestBuildTCPRoutesNoInjectionWhenTransportMissing(t *testing.T) {
	nameMap := map[string]string{"svc": "svc-1234"}
	svcInfo := map[string]tcpSvcInfo{"svc": {port: 1234, hosts: []string{"h"}, transport: ""}}
	routers := map[string]json.RawMessage{
		"r": json.RawMessage(`{"entryPoints":["ep"],"rule":"HostSNI(\"*\")","service":"svc"}`),
	}
	routes, err := buildTCPRoutes("ns", routers, nameMap, svcInfo)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	entry := getFirstServiceEntry(t, routes)
	_, has := entry["serversTransport"]
	require.False(t, has, "serversTransport should not be injected when transport missing")
}

// TestInjectServersTransportNoopShapes covers early-return branches to avoid panics and no mutation.
func TestInjectServersTransportNoopShapes(t *testing.T) {
	type testCase struct {
		name  string
		input map[string]interface{}
	}

	tests := []testCase{
		{name: "no-spec", input: map[string]interface{}{}},
		{name: "spec-no-routes", input: map[string]interface{}{"spec": map[string]interface{}{}}},
		{name: "routes-empty", input: map[string]interface{}{"spec": map[string]interface{}{"routes": []interface{}{}}}},
		{name: "first-route-not-map", input: map[string]interface{}{"spec": map[string]interface{}{"routes": []interface{}{"bad"}}}},
		{name: "route-no-services", input: map[string]interface{}{"spec": map[string]interface{}{"routes": []interface{}{map[string]interface{}{}}}}},
		{name: "service-list-empty", input: map[string]interface{}{"spec": map[string]interface{}{"routes": []interface{}{map[string]interface{}{"services": []interface{}{}}}}}},
		{name: "first-service-not-map", input: map[string]interface{}{"spec": map[string]interface{}{"routes": []interface{}{map[string]interface{}{"services": []interface{}{"bad"}}}}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := tc.input
			before := deepCopyMap(t, u)
			injectServersTransport(u, "t")
			require.Equal(t, before, u, "structure should remain unchanged for case %q", tc.name)
		})
	}
}

// deepCopyMap performs a recursive deep copy of common JSON-like Go values
// (map[string]interface{}, []interface{}, and primitive types). This avoids
// using json.Marshal/unmarshal which can change numeric types (float64) and
// doesn't preserve non-JSON-serializable values.
func deepCopyMap(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	if m == nil {
		return nil
	}
	v := deepCopyValue(m)
	if v == nil {
		return nil
	}
	if vm, ok := v.(map[string]interface{}); ok {
		return vm
	}
	t.Fatalf("deepCopyMap: expected map[string]interface{} result but got %T: %#v", v, v)
	// Unreachable: t.Fatalf calls FailNow, but the compiler requires a return
	// on this code path because it can't reason about test helpers' control flow.
	return nil
}

func deepCopyValue(v interface{}) interface{} {
	switch v := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[k] = deepCopyValue(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, val := range v {
			out[i] = deepCopyValue(val)
		}
		return out
	default:
		// Primitive types (string, bool, numbers, nil, etc.) are immutable in
		// this context so returning as-is is fine.
		return v
	}
}

// TestTransformUDPNilInput returns nils and no error.
func TestTransformUDPNilInput(t *testing.T) {
	routes, svcs, slices, err := TransformUDP(nil, "ns")
	require.NoError(t, err)
	require.Nil(t, routes)
	require.Nil(t, svcs)
	require.Nil(t, slices)
}

// TestExtractServersTransportNameInvalidJSON returns empty transport name.
func TestExtractServersTransportNameInvalidJSON(t *testing.T) {
	got := extractServersTransportName(json.RawMessage("{"))
	require.Equal(t, "", got)
}

// TestExtractLBAddressesPortURLFallback ensures we use url field when address empty.
func TestExtractLBAddressesPortURLFallback(t *testing.T) {
	raw := json.RawMessage(`{"loadBalancer":{"servers":[{"url":"http://h:3456"}]}}`)
	hosts, port, err := extractLBAddressesPort(raw)
	require.NoError(t, err)
	require.Equal(t, []string{"h"}, hosts)
	require.Equal(t, int32(3456), port)
}

// TestTransformUDPHappyPath validates a well-formed UDP config produces the expected
// routes, services and endpoint slices.
func TestTransformUDPHappyPath(t *testing.T) {
	svc := json.RawMessage(`{"loadBalancer":{"servers":[{"address":"udp-host.example.com:5005"}]}}`)
	router := json.RawMessage(`{"entryPoints":["udp-ep"],"service":"udpsvc"}`)
	cfg := &traefikconfig.Config{
		UDP: &traefikconfig.TCPUDPConfig{
			Services: map[string]json.RawMessage{"udpsvc": svc},
			Routers:  map[string]json.RawMessage{"udp-r1": router},
		},
	}

	routes, svcs, slices, err := TransformUDP(cfg, "ns")
	require.NoError(t, err)
	require.Len(t, routes, 1, "expected exactly one UDP route")
	require.Len(t, svcs, 1, "expected exactly one UDP service")
	require.Len(t, slices, 1, "expected exactly one UDP endpoint slice")

	// The service should expose the extracted port.
	require.NotEmpty(t, svcs[0].Spec.Ports)
	require.Equal(t, int32(5005), svcs[0].Spec.Ports[0].Port)
}

// TestTransformUDPMissingServiceError ensures an error is returned when a UDP router
// references a service not present in the services map.
func TestTransformUDPMissingServiceError(t *testing.T) {
	router := json.RawMessage(`{"entryPoints":["ep"],"service":"missing"}`)
	cfg := &traefikconfig.Config{
		UDP: &traefikconfig.TCPUDPConfig{
			Services: map[string]json.RawMessage{},
			Routers:  map[string]json.RawMessage{"r1": router},
		},
	}
	_, _, _, err := TransformUDP(cfg, "ns")
	require.Error(t, err)
}

// TestTransformUDPNoRouters produces no routes, but services/slices are still created.
func TestTransformUDPNoRouters(t *testing.T) {
	svc := json.RawMessage(`{"loadBalancer":{"servers":[{"address":"h:6000"}]}}`)
	cfg := &traefikconfig.Config{
		UDP: &traefikconfig.TCPUDPConfig{
			Services: map[string]json.RawMessage{"s1": svc},
			Routers:  map[string]json.RawMessage{},
		},
	}
	routes, svcs, slices, err := TransformUDP(cfg, "ns")
	require.NoError(t, err)
	require.Empty(t, routes, "no routers should produce no routes")
	require.Len(t, svcs, 1, "service should still be created")
	require.Len(t, slices, 1, "endpoint slice should still be created")
}

// TestBuildUDPServiceIndexParseError returns an error for a malformed service spec.
func TestBuildUDPServiceIndexParseError(t *testing.T) {
	_, err := buildUDPServiceIndex(map[string]json.RawMessage{
		"bad": json.RawMessage(`{"loadBalancer":{"servers":[{"address":"noport"}]}}`),
	})
	require.Error(t, err, "expected error for service without port")
}

// TestBuildUDPRoutesInvalidJSON returns an error for malformed router JSON.
func TestBuildUDPRoutesInvalidJSON(t *testing.T) {
	nameMap := map[string]string{"svc": "svc-9000"}
	_, err := buildUDPRoutes(map[string]json.RawMessage{"bad": json.RawMessage("not-json")}, nameMap, "ns")
	require.Error(t, err)
}
