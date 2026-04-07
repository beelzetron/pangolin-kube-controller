package kube

import (
	"errors"
	"testing"

	"pangolin-kube-controller/internal/config"

	"github.com/stretchr/testify/require"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestNewClientsInClusterConfigError(t *testing.T) {
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
	oldAPIExt := newApiextensionsForConfig
	defer func() {
		inClusterConfig = oldICC
		newDynamicForConfig = oldDyn
		newKubernetesForConfig = oldK8s
		newApiextensionsForConfig = oldAPIExt
	}()

	capturedCfgDyn := (*rest.Config)(nil)
	capturedCfgK8s := (*rest.Config)(nil)
	capturedCfgAPIExt := (*rest.Config)(nil)

	inClusterConfig = func() (*rest.Config, error) { return &rest.Config{}, nil }
	newDynamicForConfig = func(c *rest.Config) (*dynamic.DynamicClient, error) {
		capturedCfgDyn = c
		return nil, nil
	}
	newKubernetesForConfig = func(c *rest.Config) (*kubernetes.Clientset, error) {
		capturedCfgK8s = c
		return nil, nil
	}
	newApiextensionsForConfig = func(c *rest.Config) (apiextensionsclient.Interface, error) {
		capturedCfgAPIExt = c
		return nil, nil
	}

	cfg := &config.Config{ClientQPS: 123.0, ClientBurst: 456}
	clients, err := NewClients(cfg)
	require.NoError(t, err)
	require.NotNil(t, clients.REST, "expected REST config")
	require.Equal(t, float32(123.0), clients.REST.QPS, "QPS not applied")
	require.Equal(t, 456, clients.REST.Burst, "Burst not applied")
	require.True(t, capturedCfgDyn == clients.REST && capturedCfgK8s == clients.REST && capturedCfgAPIExt == clients.REST, "expected same *rest.Config instance passed to all constructors")
}

func TestNewClientsDefaultQPSAndBurst(t *testing.T) {
	oldICC := inClusterConfig
	oldDyn := newDynamicForConfig
	oldK8s := newKubernetesForConfig
	oldAPIExt := newApiextensionsForConfig
	defer func() {
		inClusterConfig = oldICC
		newDynamicForConfig = oldDyn
		newKubernetesForConfig = oldK8s
		newApiextensionsForConfig = oldAPIExt
	}()

	inClusterConfig = func() (*rest.Config, error) { return &rest.Config{}, nil }
	newDynamicForConfig = func(*rest.Config) (*dynamic.DynamicClient, error) { return nil, nil }
	newKubernetesForConfig = func(*rest.Config) (*kubernetes.Clientset, error) { return nil, nil }
	newApiextensionsForConfig = func(*rest.Config) (apiextensionsclient.Interface, error) { return nil, nil }

	clients, err := NewClients(&config.Config{})
	require.NoError(t, err)
	require.Equal(t, float32(20), clients.REST.QPS, "expected default QPS of 20")
	require.Equal(t, 50, clients.REST.Burst, "expected default Burst of 50")
}

func TestNewClientsDynamicForConfigError(t *testing.T) {
	oldICC := inClusterConfig
	oldDyn := newDynamicForConfig
	oldAPIExt := newApiextensionsForConfig
	defer func() {
		inClusterConfig = oldICC
		newDynamicForConfig = oldDyn
		newApiextensionsForConfig = oldAPIExt
	}()
	inClusterConfig = func() (*rest.Config, error) { return &rest.Config{}, nil }
	newDynamicForConfig = func(_ *rest.Config) (*dynamic.DynamicClient, error) { return nil, errors.New("dynerr") }
	newApiextensionsForConfig = func(*rest.Config) (apiextensionsclient.Interface, error) { return nil, nil }

	_, err := NewClients(&config.Config{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "dynamic client error")
}

func TestNewClientsKubernetesForConfigError(t *testing.T) {
	oldICC := inClusterConfig
	oldDyn := newDynamicForConfig
	oldK8s := newKubernetesForConfig
	oldAPIExt := newApiextensionsForConfig
	defer func() {
		inClusterConfig = oldICC
		newDynamicForConfig = oldDyn
		newKubernetesForConfig = oldK8s
		newApiextensionsForConfig = oldAPIExt
	}()
	inClusterConfig = func() (*rest.Config, error) { return &rest.Config{}, nil }
	newDynamicForConfig = func(*rest.Config) (*dynamic.DynamicClient, error) { return nil, nil }
	newKubernetesForConfig = func(_ *rest.Config) (*kubernetes.Clientset, error) { return nil, errors.New("k8serr") }
	newApiextensionsForConfig = func(*rest.Config) (apiextensionsclient.Interface, error) { return nil, nil }

	_, err := NewClients(&config.Config{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "clientset error")
}

func TestNewClientsApiextensionsForConfigError(t *testing.T) {
	oldICC := inClusterConfig
	oldDyn := newDynamicForConfig
	oldK8s := newKubernetesForConfig
	oldAPIExt := newApiextensionsForConfig
	defer func() {
		inClusterConfig = oldICC
		newDynamicForConfig = oldDyn
		newKubernetesForConfig = oldK8s
		newApiextensionsForConfig = oldAPIExt
	}()
	inClusterConfig = func() (*rest.Config, error) { return &rest.Config{}, nil }
	newDynamicForConfig = func(*rest.Config) (*dynamic.DynamicClient, error) { return nil, nil }
	newKubernetesForConfig = func(*rest.Config) (*kubernetes.Clientset, error) { return nil, nil }
	newApiextensionsForConfig = func(_ *rest.Config) (apiextensionsclient.Interface, error) {
		return nil, errors.New("apiextensionserr")
	}

	_, err := NewClients(&config.Config{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "apiextensions clientset error")
}
