// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"crypto/tls"
	stdlibx509 "crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/siderolabs/crypto/x509"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
)

const (
	chuboTLSRootDir = "/var/lib/chubo/certs"
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

// EnsureChuboServiceTLSMaterial ensures server-side TLS material exists on disk for the given service.
//
// The material is generated from the OS issuing CA and the API SAN resource (which includes node IPs/FQDN),
// so external clients can verify the certificate when using the advertised node IP address.
func EnsureChuboServiceTLSMaterial(ctx context.Context, r runtime.Runtime, serviceID string) error {
	paths, ok := chuboServiceTLSPaths(serviceID)
	if !ok {
		return nil
	}

	// Keep it idempotent and avoid churn across reboots: generate once, then reuse.
	if fileNonEmpty(paths.CA) && fileNonEmpty(paths.Cert) && fileNonEmpty(paths.Key) {
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

	ca, err := x509.NewCertificateAuthorityFromCertificateAndKey(rootSpec.IssuingCA)
	if err != nil {
		return fmt.Errorf("failed to parse OS issuing CA: %w", err)
	}

	kp, err := x509.NewKeyPair(ca,
		x509.IPAddresses(certSANs.StdIPs()),
		x509.DNSNames(certSANs.DNSNames),
		x509.CommonName(certSANs.FQDN),
		x509.NotAfter(time.Now().Add(x509.DefaultCertificateValidityDuration)),
		x509.KeyUsage(stdlibx509.KeyUsageDigitalSignature),
		x509.ExtKeyUsage([]stdlibx509.ExtKeyUsage{
			stdlibx509.ExtKeyUsageServerAuth,
			stdlibx509.ExtKeyUsageClientAuth,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to generate %s TLS cert: %w", serviceID, err)
	}

	certAndKey := x509.NewCertificateAndKeyFromKeyPair(kp)

	if err := writeFileAtomic(paths.CA, rootSpec.IssuingCA.Crt, 0o644); err != nil {
		return err
	}

	if err := writeFileAtomic(paths.Cert, certAndKey.Crt, 0o644); err != nil {
		return err
	}

	if err := writeFileAtomic(paths.Key, certAndKey.Key, 0o600); err != nil {
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
