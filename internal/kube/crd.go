package kube

import (
	"context"
	"fmt"

	"pangolin-kube-controller/internal/transform/config"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var traefikCRDs = []string{
	"middlewares",
	"ingressroutes",
	"traefikservices",
	"serverstransports",
	"ingressroutetcps",
	"ingressrouteudps",
	"serverstransporttcps",
}

func CheckCRDs(ctx context.Context, clientset apiextensionsclient.Interface) error {
	for _, resource := range traefikCRDs {
		name := resource + "." + config.Group
		_, err := clientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("required CRD %q not found: %w", name, err)
		}
	}
	return nil
}
