package testschema

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	grpTraefik = "traefik.io"
	verAlpha   = "v1alpha1"
)

func TestMapCRDByKind(t *testing.T) {
	crdA := &apiextensionsv1.CustomResourceDefinition{Spec: apiextensionsv1.CustomResourceDefinitionSpec{Names: apiextensionsv1.CustomResourceDefinitionNames{Kind: "Foo"}}}
	crdB := &apiextensionsv1.CustomResourceDefinition{Spec: apiextensionsv1.CustomResourceDefinitionSpec{Names: apiextensionsv1.CustomResourceDefinitionNames{Kind: "Bar"}}}
	m := MapCRDByKind([]*apiextensionsv1.CustomResourceDefinition{crdA, crdB})
	if m["Foo"] != crdA || m["Bar"] != crdB {
		t.Fatalf("unexpected map: %#v", m)
	}
}

func TestResolveSchemaNilCRD(t *testing.T) {
	if _, _, err := resolveSchema(nil, "v1alpha1"); err == nil {
		t.Fatalf("expected error for nil CRD")
	}
}

func TestResolveSchemaFallback(t *testing.T) {
	schema := &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"}}
	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group:    grpTraefik,
			Names:    apiextensionsv1.CustomResourceDefinitionNames{Kind: "IngressRoute"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: verAlpha, Served: true, Schema: schema}},
		},
	}
	vs, v, err := resolveSchema(crd, "v1beta1")
	if err != nil || vs == nil || v != verAlpha {
		t.Fatalf("unexpected resolve fallback: vs=%v v=%s err=%v", vs, v, err)
	}
}

func TestValidatorForAndLightValidate(t *testing.T) {
	schema := &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"}}
	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group:    grpTraefik,
			Names:    apiextensionsv1.CustomResourceDefinitionNames{Kind: "IngressRoute"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: verAlpha, Served: true, Schema: schema}},
		},
	}
	fn, err := ValidatorFor(crd, verAlpha)
	if err != nil || fn == nil {
		t.Fatalf("unexpected validator: fn is nil=%t err=%v", fn == nil, err)
	}
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(grpTraefik + "/" + verAlpha)
	obj.SetKind("IngressRoute")
	obj.SetName("test")
	errs := fn(obj)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidateFunction(t *testing.T) {
	// nil object
	if errs := Validate(nil, nil); len(errs) == 0 {
		t.Fatalf("expected error for nil obj")
	}
	// missing CRD
	obj := &unstructured.Unstructured{}
	obj.SetKind("X")
	if errs := Validate(obj, map[string]*apiextensionsv1.CustomResourceDefinition{}); len(errs) == 0 {
		t.Fatalf("expected error for missing CRD")
	}
	// happy path
	schema := &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"}}
	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group:    grpTraefik,
			Names:    apiextensionsv1.CustomResourceDefinitionNames{Kind: "IngressRoute"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: verAlpha, Served: true, Schema: schema}},
		},
	}
	obj = &unstructured.Unstructured{}
	obj.SetAPIVersion(grpTraefik + "/" + verAlpha)
	obj.SetKind("IngressRoute")
	obj.SetName("ok")
	errs := Validate(obj, map[string]*apiextensionsv1.CustomResourceDefinition{"IngressRoute": crd})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateAPIVersion(t *testing.T) {
	// missing apiVersion
	errs := validateAPIVersion("", "g", "v")
	if len(errs) == 0 {
		t.Fatalf("expected error for missing apiVersion")
	}
	// group mismatch
	errs = validateAPIVersion("g2/v1alpha1", "g", "v1alpha1")
	if len(errs) == 0 {
		t.Fatalf("expected group mismatch error")
	}
	// version mismatch
	errs = validateAPIVersion("g/v1beta1", "g", "v1alpha1")
	if len(errs) == 0 {
		t.Fatalf("expected version mismatch error")
	}
}

func TestResolveSchemaExactVersion(t *testing.T) {
	schemaA := &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"}}
	schemaB := &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"}}
	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: grpTraefik,
			Names: apiextensionsv1.CustomResourceDefinitionNames{Kind: "IngressRoute"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{Name: "v1beta1", Served: true, Schema: schemaB},
				{Name: verAlpha, Served: true, Schema: schemaA},
			},
		},
	}
	vs, v, err := resolveSchema(crd, "v1beta1")
	if err != nil || vs == nil || v != "v1beta1" {
		t.Fatalf("unexpected resolve exact: vs=%v v=%s err=%v", vs, v, err)
	}
}

func TestValidateErrorsForKindAndName(t *testing.T) {
	schema := &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"}}
	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group:    grpTraefik,
			Names:    apiextensionsv1.CustomResourceDefinitionNames{Kind: "IngressRoute"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: verAlpha, Served: true, Schema: schema}},
		},
	}
	fn, err := ValidatorFor(crd, verAlpha)
	if err != nil || fn == nil {
		t.Fatalf("unexpected validator: fn is nil=%t err=%v", fn == nil, err)
	}
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(grpTraefik + "/" + verAlpha)
	obj.SetKind("IngressRouteTCP") // wrong kind
	// Name left empty to trigger metadata.name error
	errs := fn(obj)
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 errors (kind mismatch and empty name), got %v", errs)
	}
}

func TestValidateAPIVersionNoExpectedVersion(t *testing.T) {
	errs := validateAPIVersion("g/v3", "g", "")
	if len(errs) != 0 {
		t.Fatalf("expected no errors when expected version is empty, got %v", errs)
	}
}

func TestResolveSchemaNoSchemaFound(t *testing.T) {
	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: grpTraefik,
			Names: apiextensionsv1.CustomResourceDefinitionNames{Kind: "IngressRoute"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{Name: verAlpha, Served: true, Schema: nil},
				{Name: "v1beta1", Served: true, Schema: &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: nil}},
			},
		},
	}
	vs, v, err := resolveSchema(crd, verAlpha)
	if err == nil || vs != nil || v != verAlpha {
		t.Fatalf("expected no schema found error; vs=%v v=%s err=%v", vs, v, err)
	}
}

func TestValidateEmptyNameError(t *testing.T) {
	schema := &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"}}
	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group:    grpTraefik,
			Names:    apiextensionsv1.CustomResourceDefinitionNames{Kind: "IngressRoute"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: verAlpha, Served: true, Schema: schema}},
		},
	}
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(grpTraefik + "/" + verAlpha)
	obj.SetKind("IngressRoute")
	// Name empty
	errs := Validate(obj, map[string]*apiextensionsv1.CustomResourceDefinition{"IngressRoute": crd})
	if len(errs) == 0 {
		t.Fatalf("expected error on empty metadata.name")
	}
}
