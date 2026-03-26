package sanitize

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	traefikconfig "pangolin-kube-controller/internal/transform/config"
)

// TestSanitizeTraefikConfigNil ensures nil input returns nil, nil.
func TestSanitizeTraefikConfigNilInput(t *testing.T) {
	t.Parallel()

	got, err := SanitizeTraefikConfig(nil)
	require.NoError(t, err)
	require.Nil(t, got)
}

// TestSanitizeTraefikConfigEmptyHTTP verifies that an empty HTTP config is
// processed without error and the output maps are initialised.
func TestSanitizeTraefikConfigEmptyHTTP(t *testing.T) {
	t.Parallel()

	cfg := &traefikconfig.Config{
		HTTP: traefikconfig.HTTPConfig{
			Middlewares:       map[string]json.RawMessage{},
			Routers:           map[string]json.RawMessage{},
			Services:          map[string]json.RawMessage{},
			ServersTransports: map[string]json.RawMessage{},
		},
	}
	got, err := SanitizeTraefikConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got.HTTP.Middlewares)
	require.Empty(t, got.HTTP.Routers)
	require.Empty(t, got.HTTP.Services)
}

// TestSanitizeTraefikConfigRenamesMiddleware verifies that middleware names are
// sanitized to Kubernetes-safe values.
func TestSanitizeTraefikConfigRenamesMiddleware(t *testing.T) {
	t.Parallel()

	cfg := &traefikconfig.Config{
		HTTP: traefikconfig.HTTPConfig{
			Middlewares: map[string]json.RawMessage{
				"my Middleware!": json.RawMessage(`{"headers":{}}`),
			},
			Routers:           map[string]json.RawMessage{},
			Services:          map[string]json.RawMessage{},
			ServersTransports: map[string]json.RawMessage{},
		},
	}
	got, err := SanitizeTraefikConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, got)

	sanitizedName := SanitizeResourceName("my Middleware!")
	require.Contains(t, got.HTTP.Middlewares, sanitizedName,
		"middleware should be stored under its sanitized name")
}

// TestSanitizeTraefikConfigRouterMiddlewareReference verifies that a router's
// middleware reference is rewritten to the sanitized middleware name.
func TestSanitizeTraefikConfigRouterMiddlewareReference(t *testing.T) {
	t.Parallel()

	const rawMiddlewareName = "my middleware"
	cfg := &traefikconfig.Config{
		HTTP: traefikconfig.HTTPConfig{
			Middlewares: map[string]json.RawMessage{
				rawMiddlewareName: json.RawMessage(`{}`),
			},
			Routers: map[string]json.RawMessage{
				"my-router": json.RawMessage(`{"rule":"Host(\"example.com\")","service":"my-svc","middlewares":["my middleware"]}`),
			},
			Services:          map[string]json.RawMessage{"my-svc": json.RawMessage(`{}`)},
			ServersTransports: map[string]json.RawMessage{},
		},
	}
	got, err := SanitizeTraefikConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, got)

	sanitizedMW := SanitizeResourceName(rawMiddlewareName)

	// Find the router entry (the router name itself is also sanitized).
	routerSanitized := SanitizeResourceName("my-router")
	routerRaw, ok := got.HTTP.Routers[routerSanitized]
	require.True(t, ok, "sanitized router should be present")

	var routerSpec map[string]interface{}
	require.NoError(t, json.Unmarshal(routerRaw, &routerSpec))

	mws, _ := routerSpec["middlewares"].([]interface{})
	require.Len(t, mws, 1)
	require.Equal(t, sanitizedMW, mws[0])
}

// TestSanitizeTraefikConfigServiceWeightedReferences verifies that a weighted
// service has its nested service names sanitized.
func TestSanitizeTraefikConfigServiceWeightedReferences(t *testing.T) {
	t.Parallel()

	const rawInnerSvc = "inner Service!"
	cfg := &traefikconfig.Config{
		HTTP: traefikconfig.HTTPConfig{
			Middlewares: map[string]json.RawMessage{},
			Routers:     map[string]json.RawMessage{},
			Services: map[string]json.RawMessage{
				"outer-svc": json.RawMessage(`{"weighted":{"services":[{"name":"inner Service!","weight":1}]}}`),
				rawInnerSvc: json.RawMessage(`{}`),
			},
			ServersTransports: map[string]json.RawMessage{},
		},
	}
	got, err := SanitizeTraefikConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, got)

	outerKey := SanitizeResourceName("outer-svc")
	outerRaw, ok := got.HTTP.Services[outerKey]
	require.True(t, ok, "outer service should be present under sanitized name")

	var outerSpec map[string]interface{}
	require.NoError(t, json.Unmarshal(outerRaw, &outerSpec))

	weighted, _ := outerSpec["weighted"].(map[string]interface{})
	services, _ := weighted["services"].([]interface{})
	require.Len(t, services, 1)

	svcEntry, _ := services[0].(map[string]interface{})
	require.Equal(t, SanitizeResourceName(rawInnerSvc), svcEntry["name"])
}

// TestSanitizeTraefikConfigMirroringSection verifies that a mirroring service
// has both its primary and mirror names sanitized.
func TestSanitizeTraefikConfigMirroringSection(t *testing.T) {
	t.Parallel()

	const primary = "primary svc"
	const mirror = "mirror svc"
	cfg := &traefikconfig.Config{
		HTTP: traefikconfig.HTTPConfig{
			Middlewares: map[string]json.RawMessage{},
			Routers:     map[string]json.RawMessage{},
			Services: map[string]json.RawMessage{
				"mirror-composite": json.RawMessage(`{"mirroring":{"service":"primary svc","mirrors":[{"name":"mirror svc","percent":10}]}}`),
				primary:            json.RawMessage(`{}`),
				mirror:             json.RawMessage(`{}`),
			},
			ServersTransports: map[string]json.RawMessage{},
		},
	}
	got, err := SanitizeTraefikConfig(cfg)
	require.NoError(t, err)

	compositeKey := SanitizeResourceName("mirror-composite")
	raw, ok := got.HTTP.Services[compositeKey]
	require.True(t, ok)

	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &spec))

	mirroring, _ := spec["mirroring"].(map[string]interface{})
	require.Equal(t, SanitizeResourceName(primary), mirroring["service"])

	mirrors, _ := mirroring["mirrors"].([]interface{})
	require.Len(t, mirrors, 1)
	m0, _ := mirrors[0].(map[string]interface{})
	require.Equal(t, SanitizeResourceName(mirror), m0["name"])
}

// TestSanitizeTraefikConfigServersTransportPassThrough verifies that
// ServersTransport entries are stored under sanitized keys but their raw
// specs are not modified.
func TestSanitizeTraefikConfigServersTransportPassThrough(t *testing.T) {
	t.Parallel()

	const transportName = "my transport"
	spec := json.RawMessage(`{"insecureSkipVerify":true}`)
	cfg := &traefikconfig.Config{
		HTTP: traefikconfig.HTTPConfig{
			Middlewares: map[string]json.RawMessage{},
			Routers:     map[string]json.RawMessage{},
			Services:    map[string]json.RawMessage{},
			ServersTransports: map[string]json.RawMessage{
				transportName: spec,
			},
		},
	}
	got, err := SanitizeTraefikConfig(cfg)
	require.NoError(t, err)

	sanitizedTransport := SanitizeResourceName(transportName)
	raw, ok := got.HTTP.ServersTransports[sanitizedTransport]
	require.True(t, ok)
	require.Equal(t, string(spec), string(raw))
}

// TestSanitizeTraefikConfigInvalidMiddlewareJSON returns an error when a
// middleware value contains invalid JSON.
func TestSanitizeTraefikConfigInvalidMiddlewareJSON(t *testing.T) {
	t.Parallel()

	cfg := &traefikconfig.Config{
		HTTP: traefikconfig.HTTPConfig{
			Middlewares: map[string]json.RawMessage{
				"bad-mw": json.RawMessage(`{not-json`),
			},
			Routers:           map[string]json.RawMessage{},
			Services:          map[string]json.RawMessage{},
			ServersTransports: map[string]json.RawMessage{},
		},
	}
	_, err := SanitizeTraefikConfig(cfg)
	require.Error(t, err)
}

// TestSanitizeTraefikConfigInvalidRouterJSON returns an error when a router
// value contains invalid JSON.
func TestSanitizeTraefikConfigInvalidRouterJSON(t *testing.T) {
	t.Parallel()

	cfg := &traefikconfig.Config{
		HTTP: traefikconfig.HTTPConfig{
			Middlewares: map[string]json.RawMessage{},
			Routers: map[string]json.RawMessage{
				"bad-router": json.RawMessage(`{not-json`),
			},
			Services:          map[string]json.RawMessage{},
			ServersTransports: map[string]json.RawMessage{},
		},
	}
	_, err := SanitizeTraefikConfig(cfg)
	require.Error(t, err)
}
