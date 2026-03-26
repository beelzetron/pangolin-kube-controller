package labels

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekube "k8s.io/client-go/kubernetes/fake"

	"pangolin-kube-controller/internal/config"
	prometheus "pangolin-kube-controller/internal/observability/metrics_prometheus"
)

// ---- resolveFromIngressClass (autodetect path) --------------------------------

func TestResolveFromIngressClassNoTraefikClasses(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset()
	// No IngressClasses at all → error.
	_, _, _, _, err := resolveFromIngressClass(context.Background(), kube, "", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no Traefik IngressClass found")
}

func TestResolveFromIngressClassNonTraefikClassIgnored(t *testing.T) {
	t.Parallel()

	// An IngressClass with a different controller should be ignored.
	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
		Spec:       v1.IngressClassSpec{Controller: "nginx.org/ingress-controller"},
	})
	_, _, _, _, err := resolveFromIngressClass(context.Background(), kube, "", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no Traefik IngressClass found")
}

func TestResolveFromIngressClassSingleTraefikClass(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "traefik",
			Labels: map[string]string{"app": "traefik"},
		},
		Spec: v1.IngressClassSpec{Controller: traefikControllerID},
	})

	k, v, name, labels, err := resolveFromIngressClass(context.Background(), kube, "", false)
	require.NoError(t, err)
	require.Equal(t, "app", k)
	require.Equal(t, "traefik", v)
	require.Equal(t, "traefik", name)
	require.NotEmpty(t, labels)
}

func TestResolveFromIngressClassMultipleTraefikClasses(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset(
		&v1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{Name: "traefik-1"},
			Spec:       v1.IngressClassSpec{Controller: traefikControllerID},
		},
		&v1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{Name: "traefik-2"},
			Spec:       v1.IngressClassSpec{Controller: traefikControllerID},
		},
	)

	_, _, _, _, err := resolveFromIngressClass(context.Background(), kube, "", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), ">1 Traefik IngressClasses found")
}

func TestResolveFromIngressClassSingleNoPreferredLabels(t *testing.T) {
	t.Parallel()

	// A Traefik IngressClass with no "app" or instance label → label pick error.
	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "traefik",
			Labels: map[string]string{"unrelated": "value"},
		},
		Spec: v1.IngressClassSpec{Controller: traefikControllerID},
	})

	_, _, _, _, err := resolveFromIngressClass(context.Background(), kube, "", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "label pick error")
}

// ---- resolveFromIngressClass (user-provided path) ----------------------------

func TestResolveFromIngressClassUserProvidedExists(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "my-traefik",
			Labels: map[string]string{"app": "traefik"},
		},
		Spec: v1.IngressClassSpec{Controller: traefikControllerID},
	})

	k, v, name, _, err := resolveFromIngressClass(context.Background(), kube, "my-traefik", true)
	require.NoError(t, err)
	require.Equal(t, "app", k)
	require.Equal(t, "traefik", v)
	require.Equal(t, "my-traefik", name)
}

func TestResolveFromIngressClassUserProvidedNotFound(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset() // empty – class doesn't exist
	_, _, _, _, err := resolveFromIngressClass(context.Background(), kube, "nonexistent", true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonexistent")
}

func TestResolveFromIngressClassUserProvidedNoLabels(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "traefik"},
		Spec:       v1.IngressClassSpec{Controller: traefikControllerID},
	})

	_, _, _, _, err := resolveFromIngressClass(context.Background(), kube, "traefik", true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "label pick error")
}

// ---- ResolveInstanceLabel ---------------------------------------------------

func TestResolveInstanceLabelAutodetectSuccess(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "traefik",
			Labels: map[string]string{"app": "mytraefik"},
		},
		Spec: v1.IngressClassSpec{Controller: traefikControllerID},
	})

	cfg := &config.Config{} // no label configured – autodetect
	mc := prometheus.NewCollector()
	err := ResolveInstanceLabel(context.Background(), kube, cfg, mc)
	require.NoError(t, err)
	require.Equal(t, "app", cfg.TraefikInstanceLabelKey)
	require.Equal(t, "mytraefik", cfg.TraefikInstanceLabelValue)
}

func TestResolveInstanceLabelAutodetectFailure(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset() // no classes
	cfg := &config.Config{}
	mc := prometheus.NewCollector()
	err := ResolveInstanceLabel(context.Background(), kube, cfg, mc)
	require.Error(t, err)
}

// ---- verifyIngressClassLabel ------------------------------------------------

func TestVerifyIngressClassLabelMatch(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "traefik",
			Labels: map[string]string{"app": "traefik"},
		},
	})
	cfg := &config.Config{
		IngressClass:              "traefik",
		TraefikInstanceLabelKey:   "app",
		TraefikInstanceLabelValue: "traefik",
	}
	err := verifyIngressClassLabel(context.Background(), kube, cfg, nil)
	require.NoError(t, err)
}

func TestVerifyIngressClassLabelMismatch(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "traefik",
			Labels: map[string]string{"app": "different-value"},
		},
	})
	cfg := &config.Config{
		IngressClass:              "traefik",
		TraefikInstanceLabelKey:   "app",
		TraefikInstanceLabelValue: "traefik",
	}
	err := verifyIngressClassLabel(context.Background(), kube, cfg, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "verify mismatch")
}

func TestVerifyIngressClassLabelMismatchWithMetrics(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "traefik",
			Labels: map[string]string{"app": "different-value"},
		},
	})
	cfg := &config.Config{
		IngressClass:              "traefik",
		TraefikInstanceLabelKey:   "app",
		TraefikInstanceLabelValue: "traefik",
	}
	mc := prometheus.NewCollector()
	err := verifyIngressClassLabel(context.Background(), kube, cfg, mc)
	require.Error(t, err)
}

func TestVerifyIngressClassLabelNotFound(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset() // class doesn't exist
	cfg := &config.Config{
		IngressClass:              "nonexistent",
		TraefikInstanceLabelKey:   "app",
		TraefikInstanceLabelValue: "traefik",
	}
	err := verifyIngressClassLabel(context.Background(), kube, cfg, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "verify failed")
}

func TestVerifyIngressClassLabelNoLabelsOnClass(t *testing.T) {
	t.Parallel()

	// Class exists but has no labels → value is empty string → mismatch.
	kube := fakekube.NewClientset(&v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "traefik"},
	})
	cfg := &config.Config{
		IngressClass:              "traefik",
		TraefikInstanceLabelKey:   "app",
		TraefikInstanceLabelValue: "traefik",
	}
	err := verifyIngressClassLabel(context.Background(), kube, cfg, nil)
	require.Error(t, err)
}

// ---- Monitor (cancellation) -------------------------------------------------

func TestMonitorCancelledContext(t *testing.T) {
	t.Parallel()

	kube := fakekube.NewClientset()
	cfg := &config.Config{
		IngressClass:              "traefik",
		TraefikInstanceLabelKey:   "app",
		TraefikInstanceLabelValue: "traefik",
		// Use a very long interval so the ticker never fires within the test.
		IngressClassLabelVerifyInterval: 24 * 3600 * 1e9, // 24 hours in nanoseconds
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := Monitor(ctx, kube, cfg, nil)
	require.NoError(t, err, "Monitor should return nil when context is cancelled")
}
