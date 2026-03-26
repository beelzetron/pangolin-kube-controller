package apply

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	logrus "github.com/sirupsen/logrus"
)

type ServiceOps struct {
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

func (o *ServiceOps) Apply(ctx context.Context, svc *corev1.Service) error {
	o.ensureManagedMeta(svc)
	if o.ReadOnly {
		logrus.Infof("[READ-ONLY] would apply Service %s", svc.Name)
		return nil
	}
	cli := o.Kube.CoreV1().Services(o.Namespace)
	existing, err := cli.Get(ctx, svc.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = cli.Create(ctx, svc, metav1.CreateOptions{})
			return err
		}
		return err
	}
	existing.Spec = svc.Spec
	existing.Labels = svc.Labels
	existing.Annotations = svc.Annotations
	_, err = cli.Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (o *ServiceOps) ensureManagedMeta(svc *corev1.Service) {
	if svc.Labels == nil {
		svc.Labels = map[string]string{}
	}
	if o.ManagedLabelKey != "" {
		svc.Labels[o.ManagedLabelKey] = o.ManagedLabelValue
	}
	if o.TraefikInstanceLabelKey != "" && o.TraefikInstanceLabelValue != "" {
		svc.Labels[o.TraefikInstanceLabelKey] = o.TraefikInstanceLabelValue
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	if o.ManagedAnnoKey != "" {
		svc.Annotations[o.ManagedAnnoKey] = o.ManagedAnnoValue
	}
}
