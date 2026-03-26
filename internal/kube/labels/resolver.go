package labels

import (
	"context"
	"fmt"
	"strings"
	"time"

	logrus "github.com/sirupsen/logrus"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"

	"pangolin-kube-controller/internal/config"
	prometheus "pangolin-kube-controller/internal/observability/metrics_prometheus"
)

const traefikControllerID = "traefik.io/ingress-controller"

// InstanceLabelKey is the common label key used to identify Traefik instances.
const InstanceLabelKey = "app.kubernetes.io/instance"

// ResolveInstanceLabel ensures cfg.TraefikInstanceLabelKey/Value are set and valid.
// If not provided via ENV/Config, it autodetects from the IngressClass labels using the rules:
// - consider only IngressClasses with spec.controller == traefik.io/ingress-controller when class is not user-provided
// - if 0 or >1 found → error
// - prefer label key "app"; else use "app.kubernetes.io/instance"; otherwise error
func ResolveInstanceLabel(ctx context.Context, kube kubernetes.Interface, cfg *config.Config, mc *prometheus.Collector) error {
	var source string
	if cfg.TraefikInstanceLabelKey != "" && cfg.TraefikInstanceLabelValue != "" {
		source = "configured"
		if err := validateKV(cfg.TraefikInstanceLabelKey, cfg.TraefikInstanceLabelValue); err != nil {
			if mc != nil {
				mc.InstanceLabelDetectFailures.Inc()
			}
			return fmt.Errorf("invalid TRAEFIK_INSTANCE_LABEL: %w", err)
		}
		logrus.WithFields(logrus.Fields{
			"source":        source,
			"label_key":     cfg.TraefikInstanceLabelKey,
			"label_value":   cfg.TraefikInstanceLabelValue,
			"ingress_class": cfg.IngressClass,
		}).Info("resolved traefik instance label")
		if mc != nil {
			mc.InstanceLabelDetectSuccess.Inc()
		}
		return nil
	}

	// Autodetect from IngressClass
	k, v, icName, labels, err := resolveFromIngressClass(ctx, kube, cfg.IngressClass, cfg.IngressClassProvided)
	if err != nil {
		if mc != nil {
			mc.InstanceLabelDetectFailures.Inc()
		}
		return err
	}
	cfg.TraefikInstanceLabelKey = k
	cfg.TraefikInstanceLabelValue = v
	cfg.IngressClass = icName // ensure chosen name stored

	logrus.WithFields(logrus.Fields{
		"source":        "autodetect",
		"label_key":     k,
		"label_value":   v,
		"ingress_class": icName,
		"labels":        labels,
	}).Info("resolved traefik instance label from IngressClass")
	if mc != nil {
		mc.InstanceLabelDetectSuccess.Inc()
	}
	return nil
}

func resolveFromIngressClass(ctx context.Context, kube kubernetes.Interface, ingressClassOpt string, userProvided bool) (key, value, className string, labelMap map[string]string, err error) {
	if userProvided {
		ic, e := kube.NetworkingV1().IngressClasses().Get(ctx, ingressClassOpt, metav1.GetOptions{})
		if e != nil {
			return "", "", "", nil, fmt.Errorf("ingressclass %q get error: %w", ingressClassOpt, e)
		}
		k, v, e := pickPreferredLabel(ic)
		if e != nil {
			return "", "", "", ic.GetLabels(), fmt.Errorf("ingressclass %q label pick error: %w", ic.Name, e)
		}
		return k, v, ic.Name, ic.GetLabels(), nil
	}

	// list all and filter by controller id
	list, e := kube.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
	if e != nil {
		return "", "", "", nil, fmt.Errorf("list ingressclasses error: %w", e)
	}
	candidates := make([]*v1.IngressClass, 0, len(list.Items))
	for i := range list.Items {
		ic := &list.Items[i]
		if ic.Spec.Controller == traefikControllerID {
			candidates = append(candidates, ic)
		}
	}
	switch len(candidates) {
	case 0:
		return "", "", "", nil, fmt.Errorf("no Traefik IngressClass found (controller=%s); set INGRESS_CLASS explicitly", traefikControllerID)
	case 1:
		k, v, e := pickPreferredLabel(candidates[0])
		if e != nil {
			return "", "", "", candidates[0].GetLabels(), fmt.Errorf("ingressclass %q label pick error: %w", candidates[0].Name, e)
		}
		return k, v, candidates[0].Name, candidates[0].GetLabels(), nil
	default:
		return "", "", "", nil, fmt.Errorf(">1 Traefik IngressClasses found (controller=%s); set INGRESS_CLASS explicitly", traefikControllerID)
	}
}

func pickPreferredLabel(ic *v1.IngressClass) (string, string, error) {
	labels := ic.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	if v, ok := labels["app"]; ok {
		vTrim := strings.TrimSpace(v)
		if vTrim != "" {
			if err := validateKV("app", vTrim); err != nil {
				return "", "", err
			}
			return "app", vTrim, nil
		}
	}
	if v, ok := labels[InstanceLabelKey]; ok {
		vTrim := strings.TrimSpace(v)
		if vTrim != "" {
			if err := validateKV(InstanceLabelKey, vTrim); err != nil {
				return "", "", err
			}
			return InstanceLabelKey, vTrim, nil
		}
	}
	return "", "", fmt.Errorf("preferred labels not found on IngressClass; expected app or %s", InstanceLabelKey)
}

func validateKV(key, value string) error {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return fmt.Errorf("empty key or value")
	}
	if errs := validation.IsQualifiedName(key); len(errs) > 0 {
		return fmt.Errorf("invalid label key %q: %s", key, strings.Join(errs, "; "))
	}
	if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
		return fmt.Errorf("invalid label value %q: %s", value, strings.Join(errs, "; "))
	}
	return nil
}

// Monitor periodically checks that the selected IngressClass still carries the resolved label key/value.
// If strict is true, a mismatch returns an error to trigger a CrashLoop.
func Monitor(ctx context.Context, kube kubernetes.Interface, cfg *config.Config, mc *prometheus.Collector) error {
	if cfg.IngressClass == "" || cfg.TraefikInstanceLabelKey == "" {
		return nil // nothing to verify
	}
	interval := cfg.IngressClassLabelVerifyInterval
	if interval <= 0 {
		interval = 3 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := verifyIngressClassLabel(ctx, kube, cfg, mc); err != nil {
				if cfg.IngressClassLabelStrict {
					return err
				}
				// non-strict: log already handled inside helper; continue loop
				continue
			}
		}
	}
}

func verifyIngressClassLabel(ctx context.Context, kube kubernetes.Interface, cfg *config.Config, mc *prometheus.Collector) error {
	ic, err := kube.NetworkingV1().IngressClasses().Get(ctx, cfg.IngressClass, metav1.GetOptions{})
	if mc != nil {
		mc.InstanceLabelLastCheck.Set(float64(time.Now().Unix()))
	}
	if err != nil {
		logrus.Errorf("instance label verify: failed to get IngressClass %q: %v", cfg.IngressClass, err)
		return fmt.Errorf("verify failed: %w", err)
	}
	labels := ic.GetLabels()
	actual := ""
	if labels != nil {
		actual = labels[cfg.TraefikInstanceLabelKey]
	}
	if strings.TrimSpace(actual) != strings.TrimSpace(cfg.TraefikInstanceLabelValue) {
		logrus.Errorf("instance label verify: mismatch on IngressClass %q: expected %s=%s, got %s", cfg.IngressClass, cfg.TraefikInstanceLabelKey, cfg.TraefikInstanceLabelValue, actual)
		if mc != nil {
			mc.InstanceLabelDetectFailures.Inc()
		}
		return fmt.Errorf("verify mismatch: %s", cfg.TraefikInstanceLabelKey)
	}
	logrus.Debugf("instance label verify: OK for IngressClass %q (%s=%s)", cfg.IngressClass, cfg.TraefikInstanceLabelKey, cfg.TraefikInstanceLabelValue)
	return nil
}
