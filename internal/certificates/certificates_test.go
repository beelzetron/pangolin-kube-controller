package certificates

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekube "k8s.io/client-go/kubernetes/fake"

	"pangolin-kube-controller/internal/config"
)

// generateTLSPair creates a self-signed certificate and returns (certPEM, keyPEM).
// The certificate has the given commonName and optional DNS SANs.
func generateTLSPair(t *testing.T, commonName string, dnsNames []string) ([]byte, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		DNSNames:     dnsNames,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certBuf := &bytes.Buffer{}
	require.NoError(t, pem.Encode(certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyBuf := &bytes.Buffer{}
	require.NoError(t, pem.Encode(keyBuf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))

	return certBuf.Bytes(), keyBuf.Bytes()
}

// makeTLSSecret creates a kubernetes.io/tls Secret with the given PEM data.
func makeTLSSecret(name, namespace string, certPEM, keyPEM []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": certPEM,
			"tls.key": keyPEM,
		},
	}
}

// ---- ControllerNamespace tests ----

func TestControllerNamespaceFallback(t *testing.T) {
	// In a test environment the service account file does not exist, so the
	// fallback value must be returned.
	ns := ControllerNamespace("default-fallback")
	require.Equal(t, "default-fallback", ns)
}

// ---- FetchAll tests ----

func TestFetchAllSingleSecretInControllerNamespace(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "example.com", []string{"example.com"})
	client := fakekube.NewClientset(makeTLSSecret("my-tls", "default", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "my-tls"}}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 1)
	r := results[0]
	require.Equal(t, "my-tls", r.CertName)
	require.Equal(t, "example.com", r.CommonName)
	require.Equal(t, "example.com", r.AltName)
	require.False(t, r.Wildcard)
	require.Equal(t, string(certPEM), r.CertFile)
	require.Equal(t, string(keyPEM), r.KeyFile)
}

func TestFetchAllSingleSecretInExplicitNamespace(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "ns-test.example.com", []string{"ns-test.example.com"})
	client := fakekube.NewClientset(makeTLSSecret("my-tls", "cert-manager", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "my-tls", Namespace: "cert-manager"}}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 1)
	require.Equal(t, "my-tls", results[0].CertName)
}

func TestFetchAllMultipleSecrets(t *testing.T) {
	cert1PEM, key1PEM := generateTLSPair(t, "a.example.com", []string{"a.example.com"})
	cert2PEM, key2PEM := generateTLSPair(t, "b.example.com", []string{"b.example.com"})
	client := fakekube.NewClientset(
		makeTLSSecret("secret-a", "ns1", cert1PEM, key1PEM),
		makeTLSSecret("secret-b", "ns2", cert2PEM, key2PEM),
	)

	refs := []config.CertificateSecretRef{
		{SecretName: "secret-a", Namespace: "ns1"},
		{SecretName: "secret-b", Namespace: "ns2"},
	}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 2)
	require.Equal(t, "secret-a", results[0].CertName)
	require.Equal(t, "secret-b", results[1].CertName)
}

func TestFetchAllNamespaceFallback(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "fallback.example.com", nil)
	// Secret is in "controller-ns", no namespace configured in ref
	client := fakekube.NewClientset(makeTLSSecret("fallback-tls", "controller-ns", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "fallback-tls"}} // no namespace
	results := FetchAll(context.Background(), client, refs, "controller-ns")

	require.Len(t, results, 1)
}

func TestFetchAllMissingSecret(t *testing.T) {
	client := fakekube.NewClientset() // empty cluster
	refs := []config.CertificateSecretRef{{SecretName: "nonexistent", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")
	require.Empty(t, results)
}

func TestFetchAllWrongSecretType(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "opaque-secret", Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")},
	}
	client := fakekube.NewClientset(secret)
	refs := []config.CertificateSecretRef{{SecretName: "opaque-secret", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")
	require.Empty(t, results)
}

func TestFetchAllMissingTLSCrt(t *testing.T) {
	_, keyPEM := generateTLSPair(t, "example.com", nil)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "no-cert", Namespace: "default"},
		Type:       corev1.SecretTypeTLS,
		Data:       map[string][]byte{"tls.key": keyPEM},
	}
	client := fakekube.NewClientset(secret)
	refs := []config.CertificateSecretRef{{SecretName: "no-cert", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")
	require.Empty(t, results)
}

func TestFetchAllMissingTLSKey(t *testing.T) {
	certPEM, _ := generateTLSPair(t, "example.com", nil)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "no-key", Namespace: "default"},
		Type:       corev1.SecretTypeTLS,
		Data:       map[string][]byte{"tls.crt": certPEM},
	}
	client := fakekube.NewClientset(secret)
	refs := []config.CertificateSecretRef{{SecretName: "no-key", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")
	require.Empty(t, results)
}

func TestFetchAllInvalidCertContent(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "bad-cert", Namespace: "default"},
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte("this is not a valid PEM certificate"),
			"tls.key": []byte("-----BEGIN EC PRIVATE KEY-----\nfakekey\n-----END EC PRIVATE KEY-----"),
		},
	}
	client := fakekube.NewClientset(secret)
	refs := []config.CertificateSecretRef{{SecretName: "bad-cert", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")
	require.Empty(t, results)
}

func TestFetchAllWildcardCertificate(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "*.example.com", []string{"*.example.com"})
	client := fakekube.NewClientset(makeTLSSecret("wildcard-tls", "default", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "wildcard-tls", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 1)
	require.True(t, results[0].Wildcard)
	require.Equal(t, "*.example.com", results[0].AltName)
	require.Equal(t, "*.example.com", results[0].CommonName)
}

func TestFetchAllNonWildcardCertificate(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "plain.example.com", []string{"plain.example.com"})
	client := fakekube.NewClientset(makeTLSSecret("plain-tls", "default", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "plain-tls", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 1)
	require.False(t, results[0].Wildcard)
}

func TestFetchAllWildcardInSAN(t *testing.T) {
	// CN is non-wildcard but a SAN has a wildcard
	certPEM, keyPEM := generateTLSPair(t, "example.com", []string{"*.example.com", "example.com"})
	client := fakekube.NewClientset(makeTLSSecret("san-wildcard", "default", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "san-wildcard", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 1)
	require.True(t, results[0].Wildcard)
	require.Equal(t, "*.example.com", results[0].AltName, "first DNS SAN should be used as altName")
}

func TestFetchAllMultipleSANsUsesFirst(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "example.com", []string{"first.example.com", "second.example.com", "third.example.com"})
	client := fakekube.NewClientset(makeTLSSecret("multi-san", "default", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "multi-san", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 1)
	require.Equal(t, "first.example.com", results[0].AltName)
}

func TestFetchAllAltNameFallsBackToCN(t *testing.T) {
	// No SANs; altName must fall back to CN
	certPEM, keyPEM := generateTLSPair(t, "cn-only.example.com", nil)
	client := fakekube.NewClientset(makeTLSSecret("cn-only", "default", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "cn-only", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 1)
	require.Equal(t, "cn-only.example.com", results[0].AltName)
}

func TestFetchAllEmptyRefsReturnsEmptySlice(t *testing.T) {
	client := fakekube.NewClientset()
	results := FetchAll(context.Background(), client, nil, "default")
	require.NotNil(t, results)
	require.Empty(t, results)
}

func TestFetchAllPartialFailureSkipsBadSecrets(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "good.example.com", []string{"good.example.com"})
	client := fakekube.NewClientset(
		makeTLSSecret("good-tls", "default", certPEM, keyPEM),
		// bad-tls doesn't exist in the fake client
	)

	refs := []config.CertificateSecretRef{
		{SecretName: "good-tls", Namespace: "default"},
		{SecretName: "missing-tls", Namespace: "default"},
	}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 1)
	require.Equal(t, "good-tls", results[0].CertName)
}

// ---- Raw PEM response format tests ----

func TestCertFileIsRawPEMNotBase64(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "pem-check.example.com", nil)
	client := fakekube.NewClientset(makeTLSSecret("pem-secret", "default", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "pem-secret", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")

	require.Len(t, results, 1)
	require.True(t, strings.HasPrefix(results[0].CertFile, "-----BEGIN "),
		"certFile must start with PEM header, got: %q", results[0].CertFile[:min(30, len(results[0].CertFile))])
	require.True(t, strings.HasPrefix(results[0].KeyFile, "-----BEGIN "),
		"keyFile must start with PEM header, got: %q", results[0].KeyFile[:min(30, len(results[0].KeyFile))])
}

// ---- No private key in logs test ----

func TestPrivateKeyNotInCertName(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "secret.example.com", nil)
	client := fakekube.NewClientset(makeTLSSecret("key-log-test", "default", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "key-log-test", Namespace: "default"}}
	results := FetchAll(context.Background(), client, refs, "default")
	require.Len(t, results, 1)

	// The CertName must be the Secret name, not any key material.
	require.Equal(t, "key-log-test", results[0].CertName)
	require.NotContains(t, results[0].CertName, "-----BEGIN")
}

// ---- HTTP Handler tests ----

func TestHandlerReturnsJSONArray(t *testing.T) {
	certPEM, keyPEM := generateTLSPair(t, "handler-test.example.com", []string{"handler-test.example.com"})
	client := fakekube.NewClientset(makeTLSSecret("handler-tls", "default", certPEM, keyPEM))

	refs := []config.CertificateSecretRef{{SecretName: "handler-tls", Namespace: "default"}}
	h := Handler(client, refs, "default")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates", http.NoBody)
	rr := httptest.NewRecorder()
	h(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp []CertificateResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	require.Equal(t, "handler-tls", resp[0].CertName)
}

func TestHandlerReturnsEmptyArrayWhenNoSecrets(t *testing.T) {
	client := fakekube.NewClientset()
	h := Handler(client, nil, "default")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/certificates", http.NoBody)
	rr := httptest.NewRecorder()
	h(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := strings.TrimSpace(rr.Body.String())
	require.Equal(t, "[]", body)
}

func TestHandlerRejectsNonGET(t *testing.T) {
	client := fakekube.NewClientset()
	h := Handler(client, nil, "default")

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/v1/certificates", http.NoBody)
		rr := httptest.NewRecorder()
		h(rr, req)
		require.Equal(t, http.StatusMethodNotAllowed, rr.Code, "method %s should be rejected", method)
	}
}
