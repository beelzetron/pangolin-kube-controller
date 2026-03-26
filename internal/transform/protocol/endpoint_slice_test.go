package protocol

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestBuildEndpointSlice(t *testing.T) {
	es, err := buildEndpointSlice("ns", "base", 8080, []string{"h1", "h2"}, corev1.ProtocolTCP)
	if err != nil {
		t.Fatalf("unexpected error building endpoint slice: %v", err)
	}
	if es.Namespace != "ns" || es.Name != "base-eps" {
		t.Fatalf("meta mismatch: %s/%s", es.Namespace, es.Name)
	}
	if len(es.Endpoints) < 2 {
		t.Fatalf("endpoints length mismatch: %+v", es.Endpoints)
	}
	if len(es.Endpoints[0].Addresses) == 0 || len(es.Endpoints[1].Addresses) == 0 {
		t.Fatalf("endpoint addresses missing: %+v", es.Endpoints)
	}
	if es.Endpoints[0].Addresses[0] != "h1" || es.Endpoints[1].Addresses[0] != "h2" {
		t.Fatalf("endpoints mismatch: %+v", es.Endpoints)
	}
	if len(es.Ports) != 1 || es.Ports[0].Port == nil || *es.Ports[0].Port != 8080 {
		t.Fatalf("ports mismatch: %+v", es.Ports)
	}
}
