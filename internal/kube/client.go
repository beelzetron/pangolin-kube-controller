package kube

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"pangolin-kube-controller/internal/config"
)

// Provide small seams for dependency injection to allow unit tests to stub
// Kubernetes client constructors without requiring a real cluster.
var (
	inClusterConfig        = rest.InClusterConfig
	newDynamicForConfig    = dynamic.NewForConfig
	newKubernetesForConfig = kubernetes.NewForConfig
)

// Clients bundles Kubernetes REST, dynamic and typed clients used by the
// controller.
type Clients struct {
	REST       *rest.Config
	Dynamic    dynamic.Interface
	Kubernetes kubernetes.Interface
}

// NewClients constructs in‑cluster clients with optional QPS/Burst overrides
// taken from cfg.
func NewClients(cfg *config.Config) (*Clients, error) {
	restCfg, err := inClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config error: %w", err)
	}
	if cfg.ClientQPS > 0 {
		restCfg.QPS = float32(cfg.ClientQPS)
	}
	if cfg.ClientBurst > 0 {
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

	return &Clients{
		REST:       restCfg,
		Dynamic:    dyn,
		Kubernetes: clientset,
	}, nil
}
