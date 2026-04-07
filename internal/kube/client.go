package kube

import (
	"fmt"

	"pangolin-kube-controller/internal/config"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Provide small seams for dependency injection to allow unit tests to stub
// Kubernetes client constructors without requiring a real cluster.
var (
	inClusterConfig           = rest.InClusterConfig
	newDynamicForConfig       = dynamic.NewForConfig
	newKubernetesForConfig    = kubernetes.NewForConfig
	newApiextensionsForConfig = func(c *rest.Config) (apiextensionsclient.Interface, error) {
		return apiextensionsclient.NewForConfig(c)
	}
)

// Clients bundles Kubernetes REST, dynamic and typed clients used by the
// controller.
type Clients struct {
	REST          *rest.Config
	Dynamic       dynamic.Interface
	Kubernetes    kubernetes.Interface
	Apiextensions apiextensionsclient.Interface
}

// NewClients constructs in‑cluster clients with optional QPS/Burst overrides
// taken from cfg. If QPS/Burst are not set in cfg, sane defaults are applied
// (20 QPS, 50 Burst) to avoid overwhelming the API server.
func NewClients(cfg *config.Config) (*Clients, error) {
	restCfg, err := inClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config error: %w", err)
	}
	// Apply sane defaults if not explicitly set
	if cfg.ClientQPS <= 0 {
		restCfg.QPS = 20
	} else {
		restCfg.QPS = float32(cfg.ClientQPS)
	}
	if cfg.ClientBurst <= 0 {
		restCfg.Burst = 50
	} else {
		restCfg.Burst = cfg.ClientBurst
	}

	dyn, err := newDynamicForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("dynamic client error: %w", err)
	}
	clientset, err := newKubernetesForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("clientset error: %w", err)
	}
	apiextensionsClientset, err := newApiextensionsForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("apiextensions clientset error: %w", err)
	}

	return &Clients{
		REST:          restCfg,
		Dynamic:       dyn,
		Kubernetes:    clientset,
		Apiextensions: apiextensionsClientset,
	}, nil
}
