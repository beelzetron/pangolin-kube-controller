package apply

import (
	"context"
	"encoding/json"
	"sort"

	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	logrus "github.com/sirupsen/logrus"
)

const (
	managedLabelKeysAnnotation = "pangolin-kube-controller/managed-label-keys"
	managedAnnoKeysAnnotation  = "pangolin-kube-controller/managed-annotation-keys"
)

type EndpointSliceOps struct {
	Kube                      kubernetes.Interface
	Namespace                 string
	ManagedLabelKey           string
	ManagedLabelValue         string
	TraefikInstanceLabelKey   string
	TraefikInstanceLabelValue string
	ManagedAnnoKey            string
	ManagedAnnoValue          string
	ReadOnly                  bool
}

func (o *EndpointSliceOps) Apply(ctx context.Context, es *discoveryv1.EndpointSlice) error {
	o.ensureManagedMeta(es)
	ensureManagedKeysTracking(es, []string{o.ManagedLabelKey, o.TraefikInstanceLabelKey}, []string{o.ManagedAnnoKey})
	if o.ReadOnly {
		logrus.Infof("[READ-ONLY] would apply EndpointSlice %s", es.Name)
		return nil
	}
	cli := o.Kube.DiscoveryV1().EndpointSlices(o.Namespace)
	existing, err := cli.Get(ctx, es.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = cli.Create(ctx, es, metav1.CreateOptions{})
			return err
		}
		return err
	}
	return o.updateExisting(ctx, existing, es)
}

func (o *EndpointSliceOps) updateExisting(ctx context.Context, existing, es *discoveryv1.EndpointSlice) error {
	existing.Endpoints = es.Endpoints
	existing.Ports = es.Ports
	existing.AddressType = es.AddressType
	if existing.Labels == nil {
		existing.Labels = map[string]string{}
	}
	if existing.Annotations == nil {
		existing.Annotations = map[string]string{}
	}

	// Remove keys previously managed by this controller, if they are no longer desired.
	removeManagedKeys(existing.Labels, es.Labels, parseManagedKeys(existing.Annotations[managedLabelKeysAnnotation]))
	removeManagedKeys(existing.Annotations, es.Annotations, parseManagedKeys(existing.Annotations[managedAnnoKeysAnnotation]))

	mergeStringMap(existing.Labels, es.Labels)
	mergeStringMap(existing.Annotations, es.Annotations)
	_, err := o.Kube.DiscoveryV1().EndpointSlices(o.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func mergeStringMap(dst, src map[string]string) {
	if src == nil {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}

func (o *EndpointSliceOps) ensureManagedMeta(es *discoveryv1.EndpointSlice) {
	if es.Labels == nil {
		es.Labels = map[string]string{}
	}
	if o.ManagedLabelKey != "" {
		es.Labels[o.ManagedLabelKey] = o.ManagedLabelValue
	}
	if o.TraefikInstanceLabelKey != "" && o.TraefikInstanceLabelValue != "" {
		es.Labels[o.TraefikInstanceLabelKey] = o.TraefikInstanceLabelValue
	}
	if es.Annotations == nil {
		es.Annotations = map[string]string{}
	}
	if o.ManagedAnnoKey != "" {
		es.Annotations[o.ManagedAnnoKey] = o.ManagedAnnoValue
	}
}

func ensureManagedKeysTracking(es *discoveryv1.EndpointSlice, labelKeys, annoKeys []string) {
	if es.Annotations == nil {
		es.Annotations = map[string]string{}
	}

	// Build a set of allowed label keys (ignore empty entries).
	allowedLabels := make(map[string]struct{}, len(labelKeys))
	for _, k := range labelKeys {
		if k == "" {
			continue
		}
		allowedLabels[k] = struct{}{}
	}
	// Only record labels that we explicitly manage.
	lbls := make(map[string]string)
	for k, v := range es.Labels {
		if _, ok := allowedLabels[k]; ok {
			lbls[k] = v
		}
	}
	es.Annotations[managedLabelKeysAnnotation] = marshalManagedKeys(lbls)

	// Exclude the tracking annotations themselves from the managed-annotation set,
	// then record only annotations we explicitly manage.
	annos := make(map[string]string, len(es.Annotations))
	for k, v := range es.Annotations {
		if k == managedLabelKeysAnnotation || k == managedAnnoKeysAnnotation {
			continue
		}
		annos[k] = v
	}
	allowedAnnos := make(map[string]struct{}, len(annoKeys))
	for _, k := range annoKeys {
		if k == "" {
			continue
		}
		allowedAnnos[k] = struct{}{}
	}
	annosFiltered := make(map[string]string)
	for k, v := range annos {
		if _, ok := allowedAnnos[k]; ok {
			annosFiltered[k] = v
		}
	}
	es.Annotations[managedAnnoKeysAnnotation] = marshalManagedKeys(annosFiltered)
}

func parseManagedKeys(raw string) map[string]struct{} {
	if raw == "" {
		return nil
	}
	var keys []string
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		logrus.Warnf("failed to parse deletion-keys annotation: %v, raw: %s", err, raw)
		return nil
	}
	out := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k == "" {
			continue
		}
		out[k] = struct{}{}
	}
	return out
}

func marshalManagedKeys(m map[string]string) string {
	if len(m) == 0 {
		return "[]"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b, err := json.Marshal(keys)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func removeManagedKeys(existing, desired map[string]string, managed map[string]struct{}) {
	if len(existing) == 0 || len(managed) == 0 {
		return
	}
	for k := range managed {
		if k == "" {
			continue
		}
		if desired == nil {
			delete(existing, k)
			continue
		}
		if _, ok := desired[k]; !ok {
			delete(existing, k)
		}
	}
}
