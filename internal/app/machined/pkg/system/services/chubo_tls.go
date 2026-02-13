// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	stdlibx509 "crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/siderolabs/crypto/x509"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
)

var (
	// chuboTLSRootDir is intentionally a var so tests can redirect writes to a temp dir.
	chuboTLSRootDir = "/var/lib/chubo/certs"
)

const (
	// chuboTLSRotateBefore controls how soon before expiry we rotate leaf certificates.
	//
	// SideroLabs crypto defaults leaf cert validity to 24h; rotate halfway through to
	// avoid churn while still keeping certs short-lived.
	chuboTLSRotateBefore = x509.DefaultCertificateValidityDuration / 2
)

type chuboTLSPaths struct {
	Dir  string
	CA   string
	Cert string
	Key  string
}

func chuboServiceTLSPaths(serviceID string) (chuboTLSPaths, bool) {
	switch serviceID {
	case OpenWontonServiceID:
		return chuboTLSPaths{
			Dir:  filepath.Join(chuboTLSRootDir, "openwonton"),
			CA:   filepath.Join(chuboTLSRootDir, "openwonton", "ca.pem"),
			Cert: filepath.Join(chuboTLSRootDir, "openwonton", "server.pem"),
			Key:  filepath.Join(chuboTLSRootDir, "openwonton", "server-key.pem"),
		}, true
	case OpenGyozaServiceID:
		return chuboTLSPaths{
			Dir:  filepath.Join(chuboTLSRootDir, "opengyoza"),
			CA:   filepath.Join(chuboTLSRootDir, "opengyoza", "ca.pem"),
			Cert: filepath.Join(chuboTLSRootDir, "opengyoza", "server.pem"),
			Key:  filepath.Join(chuboTLSRootDir, "opengyoza", "server-key.pem"),
		}, true
	default:
		return chuboTLSPaths{}, false
	}
}

func fileNonEmpty(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}

	return st.Mode().IsRegular() && st.Size() > 0
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) //nolint:errcheck

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func readPEMCertificate(path string) (*stdlibx509.Certificate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("failed to decode PEM certificate from %q", path)
	}

	cert, err := stdlibx509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from %q: %w", path, err)
	}

	return cert, nil
}

func ipAddrsToStd(ips []netip.Addr) []net.IP {
	out := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.AsSlice())
	}

	return out
}

func certHasAllIPs(cert *stdlibx509.Certificate, expected []net.IP) bool {
	for _, want := range expected {
		found := false
		for _, got := range cert.IPAddresses {
			if got.Equal(want) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func certHasAllDNSNames(cert *stdlibx509.Certificate, expected []string) bool {
	if len(expected) == 0 {
		return true
	}

	// Normalize for case-insensitive compare.
	got := make([]string, 0, len(cert.DNSNames))
	for _, n := range cert.DNSNames {
		if strings.TrimSpace(n) == "" {
			continue
		}

		got = append(got, strings.ToLower(strings.TrimRight(n, ".")))
	}

	for _, want := range expected {
		want = strings.ToLower(strings.TrimRight(strings.TrimSpace(want), "."))
		if want == "" {
			continue
		}

		if !slices.Contains(got, want) {
			return false
		}
	}

	return true
}

func chuboServiceTLSUpToDate(paths chuboTLSPaths, issuingCA *x509.PEMEncodedCertificateAndKey, certSANs *secrets.CertSANSpec) bool {
	if issuingCA == nil || len(issuingCA.Crt) == 0 {
		return false
	}

	if certSANs == nil {
		return false
	}

	if !fileNonEmpty(paths.CA) || !fileNonEmpty(paths.Cert) || !fileNonEmpty(paths.Key) {
		return false
	}

	// CA drift: if the OS issuing CA changed, rotate leaf material to match.
	if caOnDisk, err := os.ReadFile(paths.CA); err != nil || !slices.Equal(caOnDisk, issuingCA.Crt) {
		return false
	}

	cert, err := readPEMCertificate(paths.Cert)
	if err != nil {
		return false
	}

	// If the key is unreadable or doesn't match the cert, regenerate.
	keyBytes, err := os.ReadFile(paths.Key)
	if err != nil {
		return false
	}

	key, err := (&x509.PEMEncodedCertificateAndKey{Key: keyBytes}).GetKey()
	if err != nil {
		return false
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return false
	}

	// Hashi stack components don't consistently support Ed25519 private keys for
	// TLS; prefer ECDSA/RSA leaf keys even if the OS issuing CA is Ed25519.
	switch signer.(type) {
	case ed25519.PrivateKey:
		return false
	case *ecdsa.PrivateKey, *rsa.PrivateKey:
		// ok
	default:
		return false
	}

	certPubDER, err := stdlibx509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return false
	}

	keyPubDER, err := stdlibx509.MarshalPKIXPublicKey(signer.Public())
	if err != nil {
		return false
	}

	if !slices.Equal(certPubDER, keyPubDER) {
		return false
	}

	now := time.Now()
	if now.Before(cert.NotBefore) {
		return false
	}

	// Rotate if expired or close to expiry.
	if now.After(cert.NotAfter) || now.Add(chuboTLSRotateBefore).After(cert.NotAfter) {
		return false
	}

	expectedIPs := append([]net.IP(nil), ipAddrsToStd(certSANs.IPs)...)
	expectedIPs = append(expectedIPs, net.IPv4(127, 0, 0, 1), net.IPv6loopback)
	if !certHasAllIPs(cert, expectedIPs) {
		return false
	}

	if !certHasAllDNSNames(cert, certSANs.DNSNames) {
		return false
	}

	// Keep CommonName stable when the FQDN is known.
	if strings.TrimSpace(certSANs.FQDN) != "" && cert.Subject.CommonName != strings.TrimSpace(certSANs.FQDN) {
		return false
	}

	return true
}

type runtimeStateGetter interface {
	State() runtime.State
}

// EnsureChuboServiceTLSMaterial ensures server-side TLS material exists on disk for the given service.
//
// The material is generated from the OS issuing CA and the API SAN resource (which includes node IPs/FQDN),
// so external clients can verify the certificate when using the advertised node IP address.
func EnsureChuboServiceTLSMaterial(ctx context.Context, r runtimeStateGetter, serviceID string) error {
	paths, ok := chuboServiceTLSPaths(serviceID)
	if !ok {
		return nil
	}

	if err := os.MkdirAll(paths.Dir, 0o700); err != nil {
		return err
	}

	st := r.State().V1Alpha2().Resources()

	rootRes, err := safe.ReaderGet[*secrets.OSRoot](ctx, st,
		resource.NewMetadata(secrets.NamespaceName, secrets.OSRootType, secrets.OSRootID, resource.VersionUndefined))
	if err != nil {
		return err
	}

	rootSpec := rootRes.TypedSpec()
	if rootSpec.IssuingCA == nil || len(rootSpec.IssuingCA.Key) == 0 {
		return fmt.Errorf("missing OS issuing CA key (required for %s TLS)", serviceID)
	}

	certSANRes, err := safe.ReaderGet[*secrets.CertSAN](ctx, st,
		resource.NewMetadata(secrets.NamespaceName, secrets.CertSANType, secrets.CertSANAPIID, resource.VersionUndefined))
	if err != nil {
		return err
	}

	certSANs := certSANRes.TypedSpec()

	// Keep it idempotent and avoid churn across reboots: reuse existing material if it matches
	// the current OS issuing CA and the current API SAN set (which may change with networking).
	if chuboServiceTLSUpToDate(paths, rootSpec.IssuingCA, certSANs) {
		return nil
	}

	ca, err := x509.NewCertificateAuthorityFromCertificateAndKey(rootSpec.IssuingCA)
	if err != nil {
		return fmt.Errorf("failed to parse OS issuing CA: %w", err)
	}

	ips := append([]net.IP(nil), certSANs.StdIPs()...)
	// Chubo-managed services are accessed locally via 127.0.0.1/::1. Include loopback
	// SANs so internal mTLS clients can verify certificates without disabling hostname
	// verification.
	ips = append(ips, net.IPv4(127, 0, 0, 1), net.IPv6loopback)

	serialNumber, err := x509.NewSerialNumber()
	if err != nil {
		return err
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate %s ECDSA key: %w", serviceID, err)
	}

	tmpl := &stdlibx509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: strings.TrimSpace(certSANs.FQDN),
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(x509.DefaultCertificateValidityDuration),
		KeyUsage:              stdlibx509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []stdlibx509.ExtKeyUsage{stdlibx509.ExtKeyUsageServerAuth, stdlibx509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: false,
		IsCA:                  false,
		IPAddresses:           ips,
		DNSNames:              certSANs.DNSNames,
	}

	certDER, err := stdlibx509.CreateCertificate(rand.Reader, tmpl, ca.Crt, leafKey.Public(), ca.Key)
	if err != nil {
		return fmt.Errorf("failed to sign %s TLS cert: %w", serviceID, err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := stdlibx509.MarshalPKCS8PrivateKey(leafKey)
	if err != nil {
		return fmt.Errorf("failed to marshal %s TLS key: %w", serviceID, err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	// Ensure the resulting pair is parseable and matches.
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return fmt.Errorf("failed to load %s TLS keypair: %w", serviceID, err)
	}

	if err := writeFileAtomic(paths.CA, rootSpec.IssuingCA.Crt, 0o644); err != nil {
		return err
	}

	if err := writeFileAtomic(paths.Cert, certPEM, 0o644); err != nil {
		return err
	}

	if err := writeFileAtomic(paths.Key, keyPEM, 0o600); err != nil {
		return err
	}

	return nil
}

// NewChuboServiceHTTPClient creates an mTLS HTTP client for local/external access to a Chubo-managed service API.
func NewChuboServiceHTTPClient(serviceID string, timeout time.Duration) (*http.Client, error) {
	paths, ok := chuboServiceTLSPaths(serviceID)
	if !ok {
		return nil, fmt.Errorf("unknown chubo TLS service: %q", serviceID)
	}

	caPEM, err := os.ReadFile(paths.CA)
	if err != nil {
		return nil, err
	}

	pool := stdlibx509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA PEM for %s", serviceID)
	}

	cert, err := tls.LoadX509KeyPair(paths.Cert, paths.Key)
	if err != nil {
		return nil, err
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}, nil
}
