package protocol

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	logrus "github.com/sirupsen/logrus"

	"pangolin-kube-controller/internal/config"
)

type kubeServiceTarget struct {
	name      string
	namespace string
	port      int
	scheme    string
}

// ProcessServices rewrites TraefikService specs for load balancer conversions and logging.
func ProcessServices(cfg *config.Config, services map[string]json.RawMessage) map[string]json.RawMessage {
	if services == nil {
		return nil
	}
	out := make(map[string]json.RawMessage, len(services))
	for name, raw := range services {
		out[name] = processSingleService(cfg, name, raw)
	}
	return out
}

func processSingleService(cfg *config.Config, name string, raw json.RawMessage) json.RawMessage {
	trim := strings.TrimSpace(string(raw))
	if trim != "" && trim != "{}" {
		return processNonEmptyService(cfg, name, raw)
	}
	return processEmptyService(cfg, name, raw)
}

func processNonEmptyService(cfg *config.Config, name string, raw json.RawMessage) json.RawMessage {
	var spec map[string]interface{}
	if err := json.Unmarshal(raw, &spec); err == nil {
		if target, ok := convertLoadBalancerToK8sService(cfg, spec); ok {
			if b, err := json.Marshal(spec); err == nil {
				raw = b
				logrus.Infof("TraefikService %s converted to Kubernetes Service %s/%s port=%d scheme=%s", name, target.namespace, target.name, target.port, target.scheme)
			} else {
				logrus.Warnf("TraefikService %s conversion marshal failed: %v", name, err)
			}
		} else if isEmptyLoadBalancer(spec) && cfg != nil && cfg.GatewayServiceName != "" {
			if rewriteAsGatewayService(spec, cfg) {
				if b, err := json.Marshal(spec); err == nil {
					raw = b
					logrus.Infof("TraefikService %s: empty loadBalancer resolved to gateway Service %s/%s port=%d", name, cfg.Namespace, cfg.GatewayServiceName, cfg.GatewayServicePort)
				} else {
					logrus.Warnf("TraefikService %s gateway rewrite marshal failed: %v", name, err)
				}
			}
		} else if urls := extractServiceURLs(spec); len(urls) > 0 {
			logrus.Infof("TraefikService %s servers=%v", name, urls)
		}
	}
	return raw
}

func processEmptyService(cfg *config.Config, name string, raw json.RawMessage) json.RawMessage {
	urlStr := getTraefikEnvURL(cfg)
	if urlStr == "" {
		logrus.Warnf("TraefikService %s has empty spec and no derivable LB URL; resource will be invalid until populated", name)
		return raw
	}
	parsed, err := netURLParse(urlStr)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		logrus.Warnf("TraefikService %s: invalid derived LB URL '%s': %v", name, urlStr, err)
		return raw
	}
	return buildTraefikServiceSpec(urlStr)
}

func extractServiceURLs(spec map[string]interface{}) []string {
	lb, ok := spec["loadBalancer"].(map[string]interface{})
	if !ok {
		return nil
	}
	servers, ok := lb["servers"].([]interface{})
	if !ok {
		return nil
	}
	var urls []string
	for _, s := range servers {
		if m, ok := s.(map[string]interface{}); ok {
			if u, ok := m["url"].(string); ok {
				urls = append(urls, u)
			}
		}
	}
	return urls
}

func netURLParse(u string) (*url.URL, error) {
	return url.ParseRequestURI(u)
}

func buildTraefikServiceSpec(url string) json.RawMessage {
	built := map[string]interface{}{
		"loadBalancer": map[string]interface{}{
			"servers": []interface{}{map[string]interface{}{"url": url}},
		},
	}
	b, err := json.Marshal(built)
	if err != nil {
		logrus.Warnf("failed to marshal Traefik service for %s: %v", url, err)
		return []byte("{}")
	}
	logrus.Infof("Filled empty TraefikService with server %s", url)
	return b
}

func convertLoadBalancerToK8sService(cfg *config.Config, spec map[string]interface{}) (*kubeServiceTarget, bool) {
	lbServers, ok := extractLBServers(spec)
	if !ok {
		return nil, false
	}
	target, ok := parseUniformServiceTargets(lbServers)
	if !ok && cfg != nil && cfg.Namespace != "" {
		// Retry after normalizing bare in-cluster service names (e.g. "pangolin")
		// to their FQDN <name>.<namespace>.svc, per Kubernetes in-cluster DNS.
		if rewriteShortNameHosts(lbServers, cfg.Namespace) {
			target, ok = parseUniformServiceTargets(lbServers)
		}
	}
	if !ok || target == nil {
		return nil, false
	}
	serviceEntry := map[string]interface{}{
		"name":      target.name,
		"namespace": target.namespace,
		"kind":      "Service",
		"port":      target.port,
	}
	if target.scheme == "https" {
		serviceEntry["scheme"] = "https"
	}
	spec["weighted"] = map[string]interface{}{"services": []interface{}{serviceEntry}}
	delete(spec, "loadBalancer")
	return target, true
}

// rewriteShortNameHosts rewrites loadBalancer server URLs that reference a bare
// in-cluster service name (no dots, e.g. "http://pangolin:3002") into their
// FQDN form <name>.<namespace>.svc using the controller's target namespace —
// matching Kubernetes in-cluster DNS semantics (a bare name resolves in the
// pod's own namespace). Hosts that are already qualified (FQDN or IP) are left
// untouched. Returns true if every server URL used for FQDN resolution was
// rewritten (caller still runs the strict parser, which remains authoritative).
func rewriteShortNameHosts(servers []interface{}, namespace string) bool {
	if namespace == "" {
		return false
	}
	changed := false
	for _, srv := range servers {
		srvMap, ok := srv.(map[string]interface{})
		if !ok {
			return false
		}
		rawURL, ok := srvMap["url"].(string)
		if !ok || rawURL == "" {
			return false
		}
		parsed, err := netURLParse(rawURL)
		if err != nil {
			return false
		}
		host := parsed.Hostname()
		if host == "" || strings.Contains(host, ".") {
			continue // already qualified
		}
		fqdn := host + "." + namespace + ".svc"
		if p := parsed.Port(); p != "" {
			fqdn += ":" + p
		}
		srvMap["url"] = parsed.Scheme + "://" + fqdn
		changed = true
	}
	return changed
}

// isEmptyLoadBalancer reports whether the spec is a TraefikService whose
// loadBalancer carries an empty (but present) servers list — Pangolin's
// output shape for browser-gateway resources (SSH/RDP/VNC), whose backend is
// the Pangolin app Service in the controller's own namespace.
func isEmptyLoadBalancer(spec map[string]interface{}) bool {
	if len(spec) != 1 {
		return false
	}
	lb, ok := spec["loadBalancer"].(map[string]interface{})
	if !ok {
		return false
	}
	if len(lb) != 1 {
		return false
	}
	servers, ok := lb["servers"].([]interface{})
	return ok && len(servers) == 0
}

// rewriteAsGatewayService rewrites an empty-loadBalancer TraefikService into a
// Traefik v3-valid weighted service referencing the configured gateway Service
// in the controller's target namespace. This is the CRD-native equivalent of
// backing the browser-gateway route onto the Pangolin app Service.
func rewriteAsGatewayService(spec map[string]interface{}, cfg *config.Config) bool {
	if cfg == nil || cfg.GatewayServiceName == "" || cfg.Namespace == "" {
		return false
	}
	serviceEntry := map[string]interface{}{
		"name":      cfg.GatewayServiceName,
		"namespace": cfg.Namespace,
		"kind":      "Service",
		"port":      cfg.GatewayServicePort,
	}
	spec["weighted"] = map[string]interface{}{"services": []interface{}{serviceEntry}}
	delete(spec, "loadBalancer")
	return true
}

// extractLBServers validates top-level loadBalancer shape & returns servers list.
func extractLBServers(spec map[string]interface{}) ([]interface{}, bool) {
	if len(spec) != 1 {
		return nil, false
	}
	lb, ok := spec["loadBalancer"].(map[string]interface{})
	if !ok {
		return nil, false
	}
	servers, ok := lb["servers"].([]interface{})
	if !ok || len(servers) == 0 {
		return nil, false
	}
	for k := range lb { // ensure only 'servers'
		if k != "servers" {
			return nil, false
		}
	}
	return servers, true
}

// parseUniformServiceTargets ensures all server URLs point to the same k8s service target.
func parseUniformServiceTargets(servers []interface{}) (*kubeServiceTarget, bool) {
	var target *kubeServiceTarget
	for _, srv := range servers {
		srvMap, ok := srv.(map[string]interface{})
		if !ok || len(srvMap) != 1 {
			return nil, false
		}
		rawURL, ok := srvMap["url"].(string)
		if !ok || rawURL == "" {
			return nil, false
		}
		parsed, err := parseKubeServiceURL(rawURL)
		if err != nil {
			return nil, false
		}
		if target == nil {
			target = parsed
			continue
		}
		if !target.equals(parsed) {
			return nil, false
		}
	}
	return target, target != nil
}

func parseKubeServiceURL(rawURL string) (*kubeServiceTarget, error) {
	parsed, err := netURLParse(rawURL)
	if err != nil {
		return nil, err
	}
	if err := validateParsedServiceURL(parsed, rawURL); err != nil {
		return nil, err
	}
	host := parsed.Hostname()
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid service reference %q: expected <name>.<namespace>.svc", host)
	}
	if parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid service reference %q: missing name or namespace", host)
	}
	if parts[2] != "svc" {
		return nil, fmt.Errorf("invalid service reference %q: expected .svc segment", host)
	}
	name, namespace := parts[0], parts[1]
	port := derivePort(parsed)
	return &kubeServiceTarget{name: name, namespace: namespace, port: port, scheme: parsed.Scheme}, nil
}

// validateParsedServiceURL performs structural validation for a service style URL.
func validateParsedServiceURL(parsed *url.URL, raw string) error {
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %s", parsed.Scheme)
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" {
		return fmt.Errorf("unsupported path or query components in %s", raw)
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("missing host")
	}
	if !strings.Contains(host, ".svc.") && !strings.HasSuffix(host, ".svc") {
		return fmt.Errorf("host %s is not a Kubernetes service FQDN", host)
	}
	return nil
}

// derivePort returns a port for a parsed service URL.
func derivePort(parsed *url.URL) int {
	if p := parsed.Port(); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			return val
		}
	}
	if parsed.Scheme == "https" {
		return 443
	}
	return 80
}

func (t *kubeServiceTarget) equals(other *kubeServiceTarget) bool {
	if t == nil || other == nil {
		return false
	}
	return t.name == other.name && t.namespace == other.namespace && t.port == other.port && t.scheme == other.scheme
}

func getTraefikEnvURL(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	if cfg.TraefikLBURL != "" {
		return cfg.TraefikLBURL
	}
	if cfg.TraefikLBIP == "" {
		return ""
	}
	url := cfg.TraefikLBScheme + "://" + cfg.TraefikLBIP
	if cfg.TraefikLBPort != "" {
		url += ":" + cfg.TraefikLBPort
	}
	return url
}
