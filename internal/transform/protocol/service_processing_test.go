package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"pangolin-kube-controller/internal/config"
)

// Tests from controller_services_test.go

func TestProcessSingleServiceDefaultFill(t *testing.T) {
	cfg := &config.Config{}
	cfg.TraefikLBURL = "http://1.2.3.4:8080" // NOSONAR

	out := processSingleService(cfg, "svc1", json.RawMessage(`{}`))
	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &spec))
	lb := spec["loadBalancer"].(map[string]interface{})
	servers := lb["servers"].([]interface{})
	server0 := servers[0].(map[string]interface{})
	// NOSONAR: Using a hardcoded IP in tests is acceptable and not a security issue.
	require.Equal(t, "http://1.2.3.4:8080", server0["url"]) // NOSONAR
}

func TestProcessSingleServicePassthrough(t *testing.T) {
	cfg := &config.Config{}
	// NOSONAR: Using a hardcoded IP in tests is acceptable and not a security issue.
	in := json.RawMessage(`{"loadBalancer":{"servers":[{"url":"http://10.0.0.1"}]}}`) // NOSONAR
	out := processSingleService(cfg, "svc1", in)
	require.Equal(t, string(in), string(out))
}

func TestProcessSingleServiceConvertsKubernetesService(t *testing.T) {
	cfg := &config.Config{}
	in := json.RawMessage(`{"loadBalancer":{"servers":[{"url":"http://echoserver.dummyservices.svc.cluster.local:80"}]}}`)

	out := processSingleService(cfg, "svc1", in)

	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &spec))
	_, hasLB := spec["loadBalancer"]
	require.False(t, hasLB)

	weighted, ok := spec["weighted"].(map[string]interface{})
	require.True(t, ok)
	services, ok := weighted["services"].([]interface{})
	require.True(t, ok)
	require.Len(t, services, 1)
	entry, ok := services[0].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "echoserver", entry["name"])
	require.Equal(t, "dummyservices", entry["namespace"])
	require.Equal(t, "Service", entry["kind"])
	require.EqualValues(t, 80, entry["port"])
}

func TestProcessSingleServiceSkipsKubernetesServiceWithPath(t *testing.T) {
	cfg := &config.Config{}
	in := json.RawMessage(`{"loadBalancer":{"servers":[{"url":"http://echoserver.dummyservices.svc.cluster.local/api"}]}}`)

	out := processSingleService(cfg, "svc1", in)

	require.JSONEq(t, string(in), string(out))
}

func TestProcessSingleServiceSkipsKubernetesServiceWithQuery(t *testing.T) {
	cfg := &config.Config{}
	in := json.RawMessage(`{"loadBalancer":{"servers":[{"url":"http://echoserver.dummyservices.svc.cluster.local?env=dev"}]}}`)

	out := processSingleService(cfg, "svc1", in)

	require.JSONEq(t, string(in), string(out))
}

func TestExtractServiceURLs(t *testing.T) {
	spec := map[string]interface{}{
		"loadBalancer": map[string]interface{}{
			"servers": []interface{}{map[string]interface{}{"url": "http://10.0.0.1"}, map[string]interface{}{"url": "http://10.0.0.2"}}, // NOSONAR
		},
	}
	urls := extractServiceURLs(spec)
	require.ElementsMatch(t, []string{"http://10.0.0.1", "http://10.0.0.2"}, urls) // NOSONAR
}

// Tests from controller_services_more_test.go

func TestProcessEmptyServiceWithAndWithoutLBURL(t *testing.T) {
	in := json.RawMessage([]byte("{}"))
	out := processEmptyService(&config.Config{}, "svc", in)
	require.Equal(t, string(in), string(out), "without LB URL, raw should be unchanged")

	out2 := processEmptyService(&config.Config{TraefikLBURL: "http://lb:8080"}, "svc", in)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(out2, &m), "unmarshal should succeed")
	lb, ok := m["loadBalancer"].(map[string]interface{})
	require.True(t, ok, "missing or invalid loadBalancer")
	srvs, ok := lb["servers"].([]interface{})
	require.True(t, ok, "missing servers slice")
	require.GreaterOrEqual(t, len(srvs), 1, "expected at least one service")
	firstMap, ok := srvs[0].(map[string]interface{})
	require.True(t, ok, "service[0] is not a map[string]interface{}")
	urlVal, ok := firstMap["url"].(string)
	require.True(t, ok, "service[0].url missing or not a string")
	require.Equal(t, "http://lb:8080", urlVal)
}

func TestProcessNonEmptyServiceConvertToK8sService(t *testing.T) {
	in := json.RawMessage([]byte(`{"loadBalancer":{"servers":[{"url":"http://svc.ns.svc:80"}]}}`))
	out := processNonEmptyService(&config.Config{}, "svc", in)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &m), "unmarshal should succeed")
	require.Contains(t, m, "weighted")
	require.IsType(t, map[string]interface{}{}, m["weighted"], "weighted should be a map[string]interface{}")
	require.NotContains(t, m, "loadBalancer")
}

func TestParseKubeServiceURLInvalidCases(t *testing.T) {
	_, err := parseKubeServiceURL("")
	require.Error(t, err, "empty url should error")
	_, err = parseKubeServiceURL("ftp://host")
	require.Error(t, err, "unsupported scheme should error")
	_, err = parseKubeServiceURL("http://host/path")
	require.Error(t, err, "path not allowed")
	_, err = parseKubeServiceURL("http://host")
	require.Error(t, err, "not a k8s service FQDN")
	_, err = parseKubeServiceURL("http://name..svc")
	require.Error(t, err, "invalid host segments")
}
