package protocol

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	testschema "pangolin-kube-controller/internal/testschema"
	traefikconfig "pangolin-kube-controller/internal/transform/config"
)

const defaultCRDVersion = "v3.5.0"

// TestTransformTCPMissingServicePortError ensures an error is returned when
// a TCP router references a dynamic service name that has no port suffix (because
// the service wasn't defined) and thus port derivation fails.
func TestTransformTCPMissingServicePortError(t *testing.T) {
	// Router references service "missing" which isn't present in Services map.
	raw := json.RawMessage(`{"entryPoints":["ep"],"rule":"HostSNI(\"example.com\")","service":"missing"}`)
	cfg := &traefikconfig.Config{TCP: &traefikconfig.TCPUDPConfig{Routers: map[string]json.RawMessage{"r1": raw}, Services: map[string]json.RawMessage{}}}
	_, _, _, err := TransformTCP(cfg, "ns")
	require.Error(t, err, "expected error deriving port")
	require.Contains(t, strings.ToLower(err.Error()), "derive port")
}

// TestTransformRouterTCPInvalidPortSuffix exercises error path when dynToKube mapping
// contains an invalid port suffix (out of valid range) causing parsePort32 failure.
func TestTransformRouterTCPInvalidPortSuffix(t *testing.T) {
	dynToKube := map[string]string{"svc": "svc-70000"} // invalid port (>65535)
	raw := []byte(`{"entryPoints":["ep"],"rule":"HostSNI(\"ex.com\")","service":"svc"}`)
	_, _, err := transformRouterTCPToIngressRouteTCP("ns", "r1", raw, dynToKube)
	require.Error(t, err, "expected error for invalid port suffix")
	errMsgLower := strings.ToLower(err.Error())
	require.True(t, strings.Contains(errMsgLower, "parse") || strings.Contains(errMsgLower, "port") || strings.Contains(errMsgLower, "suffix"),
		"error does not contain expected keywords (parse/port/suffix): %q", err)
}

// helper to create and transform a basic TCP config used by multiple small tests
func makeBasicTCPTransform(t *testing.T) (routes []map[string]interface{}, svcs []*corev1.Service, slices []*discoveryv1.EndpointSlice) {
	t.Helper()
	// Dynamic TCP service with one server and serversTransport
	svc := json.RawMessage(`{"loadBalancer":{"servers":[{"address":"example.com:9000"}]},"serversTransport":"my-transport"}`)
	router := json.RawMessage(`{"entryPoints":["ep"],"rule":"HostSNI(\"example.com\")","service":"dyn"}`)
	cfg := &traefikconfig.Config{TCP: &traefikconfig.TCPUDPConfig{Services: map[string]json.RawMessage{"dyn": svc}, Routers: map[string]json.RawMessage{"r1": router}}}
	routesI, svcsI, slicesI, err := TransformTCP(cfg, "ns")
	require.NoError(t, err)
	return routesI, svcsI, slicesI
}

func TestTransformTCPPositiveCounts(t *testing.T) {
	routes, svcs, slices := makeBasicTCPTransform(t)
	require.Len(t, routes, 1, "expected exactly one route")
	require.Len(t, svcs, 1, "expected exactly one service")
	require.Len(t, slices, 1, "expected exactly one slice")
}

func TestTransformTCPPositiveServicePort(t *testing.T) {
	_, svcs, _ := makeBasicTCPTransform(t)
	require.NotEmpty(t, svcs, "expected at least one service")
	require.NotEmpty(t, svcs[0].Spec.Ports, "expected service to have ports")
	require.Equal(t, int32(9000), svcs[0].Spec.Ports[0].Port, "expected service port 9000")
}

// helper to extract first service entry map from the transformed routes
func getFirstServiceEntry(t *testing.T, routes []map[string]interface{}) map[string]interface{} {
	t.Helper()
	require.NotEmpty(t, routes, "no routes present")
	root := routes[0]
	specRaw, ok := root["spec"]
	require.True(t, ok, "spec key missing in route: %#v", root)
	spec, ok := specRaw.(map[string]interface{})
	require.True(t, ok, "spec not map[string]interface{}: %T %#v", specRaw, specRaw)
	routesRaw, ok := spec["routes"]
	require.True(t, ok, "routes key missing in spec: %#v", spec)
	innerRoutes, ok := routesRaw.([]interface{})
	require.True(t, ok && len(innerRoutes) > 0, "routes not []interface{} or empty: %T %#v", routesRaw, routesRaw)
	firstRaw := innerRoutes[0]
	first, ok := firstRaw.(map[string]interface{})
	require.True(t, ok, "first route entry not map: %T %#v", firstRaw, firstRaw)
	svcEntriesRaw, ok := first["services"]
	require.True(t, ok, "services key missing in first route: %#v", first)
	svcEntries, ok := svcEntriesRaw.([]interface{})
	require.True(t, ok && len(svcEntries) > 0, "services not []interface{} or empty: %T %#v", svcEntriesRaw, svcEntriesRaw)
	entryRaw := svcEntries[0]
	entry, ok := entryRaw.(map[string]interface{})
	require.True(t, ok, "service entry not map: %T %#v", entryRaw, entryRaw)
	return entry
}

func TestTransformTCPPositiveRouteServersTransport(t *testing.T) {
	routes, _, _ := makeBasicTCPTransform(t)
	entry := getFirstServiceEntry(t, routes)
	stRaw, ok := entry["serversTransport"]
	require.True(t, ok, "serversTransport key missing: %#v", entry)
	st, ok := stRaw.(string)
	require.True(t, ok && st != "", "serversTransport not string/non-empty: %T %#v", stRaw, stRaw)
}

func TestTransformTCPPositiveRoutePort(t *testing.T) {
	routes, _, _ := makeBasicTCPTransform(t)
	entry := getFirstServiceEntry(t, routes)
	portRaw, ok := entry["port"]
	require.True(t, ok, "port key missing: %#v", entry)
	portNum, ok := portRaw.(float64) // JSON numbers decode to float64
	require.True(t, ok, "port not float64: %T %#v", portRaw, portRaw)
	require.Equal(t, 9000, int(portNum), "expected route port 9000 (raw=%#v)", portRaw)
}

// Tests from tcp_conversion_test.go

func TestTCPTransformMinimal(t *testing.T) {
	b, err := os.ReadFile(testschema.TestDataPath("traefik-configs", defaultCRDVersion, "extended.json"))
	require.NoError(t, err)
	var cfg traefikconfig.Config
	require.NoError(t, json.Unmarshal(b, &cfg))

	routes, svcs, slices, err := TransformTCP(&cfg, "pangolin")
	require.NoError(t, err)
	// Expect at least one service and one route
	require.NotEmpty(t, svcs)
	require.NotEmpty(t, routes)
	// Validate IngressRouteTCP schema
	version := os.Getenv("TRAEFIK_CRD_VERSION")
	if version == "" {
		version = defaultCRDVersion
	}
	crds, err := testschema.LoadTraefikCRDs(version)
	require.NoError(t, err)
	crdMap := testschema.MapCRDByKind(crds)
	u := &unstructured.Unstructured{Object: routes[0]}
	require.Len(t, testschema.Validate(u, crdMap), 0)
	// Spot-check that service name is referenced
	spec, _ := u.UnstructuredContent()["spec"].(map[string]interface{})
	rts, _ := spec["routes"].([]interface{})
	require.NotEmpty(t, rts)
	r0, _ := rts[0].(map[string]interface{})
	ss, _ := r0["services"].([]interface{})
	require.NotEmpty(t, ss)
	ref, _ := ss[0].(map[string]interface{})
	require.NotEmpty(t, ref["name"])
	var portF float64
	if v, ok := ref["port"].(float64); ok {
		portF = v
	} else if vi, ok := ref["port"].(int); ok {
		portF = float64(vi)
	}
	require.Greater(t, int(portF), 0)
	_ = slices // kept for future content checks
}

// Tests from tcp_udp_pending_test.go

// Sketched tests for ServersTransport + TCP/UDP conversions based on locked rules.
// These are placeholders to guide upcoming implementation; enable once features land.

func TestGenerateServersTransportFromHTTPServicesPending(t *testing.T) {
	t.Skip("TODO: implement: generate ServersTransport for each referenced serversTransport and reference from TraefikService")
}

func TestTCPConversionPending(t *testing.T) {
	t.Skip("TODO: implement: TCP upstreams -> headless Service + EndpointSlice per port; IngressRouteTCP references Service/port; optional ServersTransportTCP")
}

func TestUDPConversionPending(t *testing.T) {
	t.Skip("TODO: implement: UDP upstreams -> headless Service + EndpointSlice per port; IngressRouteUDP references Service/port; normalize address scheme")
}
