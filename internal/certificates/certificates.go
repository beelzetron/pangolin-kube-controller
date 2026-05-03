package certificates

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"

	logrus "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"pangolin-kube-controller/internal/config"
)

const (
	serviceAccountNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// CertificateResponse is the JSON object returned per configured Secret by the
// GET /api/v1/certificates endpoint.
type CertificateResponse struct {
	Wildcard   bool   `json:"wildcard"`
	AltName    string `json:"altName"`
	CertName   string `json:"certName"`
	CommonName string `json:"commonName"`
	CertFile   string `json:"certFile"`
	KeyFile    string `json:"keyFile"`
}

// ControllerNamespace returns the namespace the controller Pod is running in.
// It reads from the service account namespace file first, then falls back to
// the provided fallback value.
func ControllerNamespace(fallback string) string {
	b, err := os.ReadFile(serviceAccountNamespaceFile)
	if err == nil {
		if ns := strings.TrimSpace(string(b)); ns != "" {
			return ns
		}
	}
	return fallback
}

// FetchAll reads each configured Secret reference from the Kubernetes API and
// returns decoded TLS certificate data. Secrets that are missing, have the
// wrong type, or contain unparseable certificates are skipped with an error
// log. The returned slice is never nil; an empty slice is returned when no
// valid certificates are available.
func FetchAll(ctx context.Context, kubeClient kubernetes.Interface, refs []config.CertificateSecretRef, controllerNamespace string) []CertificateResponse {
	results := make([]CertificateResponse, 0, len(refs))
	for _, ref := range refs {
		ns := ref.Namespace
		if ns == "" {
			ns = controllerNamespace
		}
		resp, ok := fetchOne(ctx, kubeClient, ref.SecretName, ns)
		if ok {
			results = append(results, resp)
		}
	}
	return results
}

func fetchOne(ctx context.Context, kubeClient kubernetes.Interface, name, namespace string) (CertificateResponse, bool) {
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("certificates: failed to get Secret %s/%s: %v", namespace, name, err)
		return CertificateResponse{}, false
	}

	if secret.Type != corev1.SecretTypeTLS {
		logrus.Errorf("certificates: Secret %s/%s is not type %s (got %s), skipping", namespace, name, corev1.SecretTypeTLS, secret.Type)
		return CertificateResponse{}, false
	}

	certPEM, ok := secret.Data["tls.crt"]
	if !ok || len(certPEM) == 0 {
		logrus.Errorf("certificates: Secret %s/%s missing tls.crt, skipping", namespace, name)
		return CertificateResponse{}, false
	}

	keyPEM, ok := secret.Data["tls.key"]
	if !ok || len(keyPEM) == 0 {
		logrus.Errorf("certificates: Secret %s/%s missing tls.key, skipping", namespace, name)
		return CertificateResponse{}, false
	}

	cert, err := parseCertificate(certPEM)
	if err != nil {
		logrus.Errorf("certificates: Secret %s/%s tls.crt cannot be parsed as X.509: %v, skipping", namespace, name, err)
		return CertificateResponse{}, false
	}

	commonName := cert.Subject.CommonName
	altName := firstDNSSAN(cert)
	if altName == "" {
		altName = commonName
	}

	return CertificateResponse{
		Wildcard:   isWildcard(cert),
		AltName:    altName,
		CertName:   name,
		CommonName: commonName,
		CertFile:   string(certPEM),
		KeyFile:    string(keyPEM),
	}, true
}

func parseCertificate(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in tls.crt")
	}
	return x509.ParseCertificate(block.Bytes)
}

func firstDNSSAN(cert *x509.Certificate) string {
	if len(cert.DNSNames) > 0 {
		return cert.DNSNames[0]
	}
	return ""
}

func isWildcard(cert *x509.Certificate) bool {
	if strings.HasPrefix(cert.Subject.CommonName, "*.") {
		return true
	}
	for _, san := range cert.DNSNames {
		if strings.HasPrefix(san, "*.") {
			return true
		}
	}
	return false
}

// Handler returns an http.HandlerFunc for GET /api/v1/certificates. It reads
// each configured Secret from the Kubernetes API and returns decoded raw PEM
// certificate data as a JSON array. Private key material is never logged.
func Handler(kubeClient kubernetes.Interface, refs []config.CertificateSecretRef, controllerNamespace string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		certs := FetchAll(r.Context(), kubeClient, refs, controllerNamespace)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(certs); err != nil {
			logrus.Errorf("certificates: failed to encode response: %v", err)
		}
	}
}
