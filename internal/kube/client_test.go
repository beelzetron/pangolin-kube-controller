package kube

import (
	"errors"
	"testing"

	"pangolin-kube-controller/internal/config"

	"github.com/stretchr/testify/require"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestNewClientsInClusterConfigError(t *testing.T) {
	// Stub inClusterConfig to ensure deterministic error in any environment
	oldICC := inClusterConfig
	inClusterConfig = func() (*rest.Config, error) { return nil, errors.New("boom") }
	t.Cleanup(func() { inClusterConfig = oldICC })

	_, err := NewClients(&config.Config{})
	require.Error(t, err, "expected error from InClusterConfig in unit test env")
	require.Contains(t, err.Error(), "in-cluster config error")
}

func TestNewClientsAppliesQPSAndBurst(t *testing.T) {
	oldICC := inClusterConfig
	oldDyn := newDynamicForConfig
	oldK8s := newKubernetesForConfig
	defer func() {
		inClusterConfig = oldICC
		newDynamicForConfig = oldDyn
		newKubernetesForConfig = oldK8s
	}()

	capturedCfgDyn := (*rest.Config)(nil)
	capturedCfgK8s := (*rest.Config)(nil)

	inClusterConfig = func() (*rest.Config, error) { return &rest.Config{}, nil }
	newDynamicForConfig = func(c *rest.Config) (*dynamic.DynamicClient, error) {
		capturedCfgDyn = c
		return nil, nil
	}
	newKubernetesForConfig = func(c *rest.Config) (*kubernetes.Clientset, error) {
		capturedCfgK8s = c
		return nil, nil
	}

	cfg := &config.Config{ClientQPS: 123.0, ClientBurst: 456}
	clients, err := NewClients(cfg)
	require.NoError(t, err)
	require.NotNil(t, clients.REST, "expected REST config")
	require.Equal(t, float32(123.0), clients.REST.QPS, "QPS not applied")
	require.Equal(t, 456, clients.REST.Burst, "Burst not applied")
	require.True(t, capturedCfgDyn == clients.REST && capturedCfgK8s == clients.REST, "expected same *rest.Config instance passed to all constructors")
}

func TestNewClientsDynamicForConfigError(t *testing.T) {
	oldICC := inClusterConfig
	oldDyn := newDynamicForConfig
	defer func() {
		inClusterConfig = oldICC
		newDynamicForConfig = oldDyn
	}()
	inClusterConfig = func() (*rest.Config, error) { return &rest.Config{}, nil }
	newDynamicForConfig = func(_ *rest.Config) (*dynamic.DynamicClient, error) { return nil, errors.New("dynerr") }

	_, err := NewClients(&config.Config{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "dynamic client error")
}

func TestNewClientsKubernetesForConfigError(t *testing.T) {
	oldICC := inClusterConfig
	oldDyn := newDynamicForConfig
	oldK8s := newKubernetesForConfig
	defer func() {
		inClusterConfig = oldICC
		newDynamicForConfig = oldDyn
		newKubernetesForConfig = oldK8s
	}()
	inClusterConfig = func() (*rest.Config, error) { return &rest.Config{}, nil }
	newDynamicForConfig = func(*rest.Config) (*dynamic.DynamicClient, error) { return nil, nil }
	newKubernetesForConfig = func(_ *rest.Config) (*kubernetes.Clientset, error) { return nil, errors.New("k8serr") }

	_, err := NewClients(&config.Config{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "clientset error")
}
