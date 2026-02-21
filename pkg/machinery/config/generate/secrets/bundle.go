// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package secrets

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/siderolabs/crypto/x509"
	"go.yaml.in/yaml/v4"

	"github.com/chubo-dev/chubo/pkg/machinery/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config/internal/cis"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/role"
)

// NewBundle creates secrets bundle generating all secrets.
func NewBundle(clock Clock, versionContract *config.VersionContract) (*Bundle, error) {
	bundle := &Bundle{
		Clock: clock,
	}

	err := bundle.populate(versionContract)
	if err != nil {
		return nil, err
	}

	return bundle, nil
}

// LoadBundle loads secrets bundle from the given file.
func LoadBundle(path string) (*Bundle, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close() //nolint: errcheck

	bundle := &Bundle{
		Clock: NewClock(),
	}

	decoder := yaml.NewDecoder(f)
	if err = decoder.Decode(&bundle); err != nil {
		return nil, err
	}

	return bundle, nil
}

// NewBundleFromConfig creates secrets bundle using existing config.
func NewBundleFromConfig(clock Clock, c config.Config) *Bundle {
	certs := &Certs{
		OS: c.Machine().Security().IssuingCA(),
	}

	cluster := &Cluster{
		ID:     c.Cluster().ID(),
		Secret: c.Cluster().Secret(),
	}

	trustd := &TrustdInfo{
		Token: c.Machine().Security().Token(),
	}

	return &Bundle{
		Clock:      clock,
		Cluster:    cluster,
		TrustdInfo: trustd,
		Certs:      certs,
	}
}

// populate fills all the missing fields in the secrets bundle.
//
//nolint:gocyclo,cyclop
func (bundle *Bundle) populate(versionContract *config.VersionContract) error {
	if bundle.Clock == nil {
		bundle.Clock = NewClock()
	}

	if bundle.Certs == nil {
		bundle.Certs = &Certs{}
	}

	if bundle.Certs.OS == nil {
		talosCA, err := NewTalosCA(bundle.Clock.Now())
		if err != nil {
			return err
		}

		bundle.Certs.OS = &x509.PEMEncodedCertificateAndKey{
			Crt: talosCA.CrtPEM,
			Key: talosCA.KeyPEM,
		}
	}

	if bundle.Secrets == nil {
		bundle.Secrets = &Secrets{}
	}

	if bundle.Secrets.BootstrapToken == "" {
		token, err := genToken(6, 16)
		if err != nil {
			return err
		}

		bundle.Secrets.BootstrapToken = token
	}

	if versionContract.Greater(config.ChuboVersion1_2) {
		if bundle.Secrets.SecretboxEncryptionSecret == "" {
			secretboxEncryptionSecret, err := cis.CreateEncryptionToken()
			if err != nil {
				return err
			}

			bundle.Secrets.SecretboxEncryptionSecret = secretboxEncryptionSecret
		}
	} else {
		if bundle.Secrets.AESCBCEncryptionSecret == "" {
			aesCBCEncryptionSecret, err := cis.CreateEncryptionToken()
			if err != nil {
				return err
			}

			bundle.Secrets.AESCBCEncryptionSecret = aesCBCEncryptionSecret
		}
	}

	if bundle.TrustdInfo == nil {
		bundle.TrustdInfo = &TrustdInfo{}
	}

	if bundle.TrustdInfo.Token == "" {
		token, err := genToken(6, 16)
		if err != nil {
			return err
		}

		bundle.TrustdInfo.Token = token
	}

	if bundle.Cluster == nil {
		bundle.Cluster = &Cluster{}
	}

	if bundle.Cluster.ID == "" {
		clusterID, err := randBytes(constants.DefaultClusterIDSize)
		if err != nil {
			return fmt.Errorf("failed to generate cluster ID: %w", err)
		}

		bundle.Cluster.ID = base64.URLEncoding.EncodeToString(clusterID)
	}

	if bundle.Cluster.Secret == "" {
		clusterSecret, err := randBytes(constants.DefaultClusterSecretSize)
		if err != nil {
			return fmt.Errorf("failed to generate cluster secret: %w", err)
		}

		bundle.Cluster.Secret = base64.StdEncoding.EncodeToString(clusterSecret)
	}

	return nil
}

// GenerateChuboAPIClientCertificate generates the admin certificate.
func (bundle *Bundle) GenerateChuboAPIClientCertificate(roles role.Set) (*x509.PEMEncodedCertificateAndKey, error) {
	return bundle.GenerateChuboAPIClientCertificateWithTTL(roles, constants.ChuboAPIDefaultCertificateValidityDuration)
}

// GenerateChuboAPIClientCertificateWithTTL generates the admin certificate with specified TTL.
func (bundle *Bundle) GenerateChuboAPIClientCertificateWithTTL(roles role.Set, crtTTL time.Duration) (*x509.PEMEncodedCertificateAndKey, error) {
	return NewAdminCertificateAndKey(
		bundle.Clock.Now(),
		bundle.Certs.OS,
		roles,
		crtTTL,
	)
}

// GenerateTalosAPIClientCertificate is a legacy alias kept for compatibility.
func (bundle *Bundle) GenerateTalosAPIClientCertificate(roles role.Set) (*x509.PEMEncodedCertificateAndKey, error) {
	return bundle.GenerateChuboAPIClientCertificate(roles)
}

// GenerateTalosAPIClientCertificateWithTTL is a legacy alias kept for compatibility.
func (bundle *Bundle) GenerateTalosAPIClientCertificateWithTTL(roles role.Set, crtTTL time.Duration) (*x509.PEMEncodedCertificateAndKey, error) {
	return bundle.GenerateChuboAPIClientCertificateWithTTL(roles, crtTTL)
}

// Validate the bundle.
//
//nolint:gocyclo,cyclop
func (bundle *Bundle) Validate() error {
	var multiErr error

	if bundle.Cluster == nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("cluster is required"))
	} else {
		if bundle.Cluster.ID == "" {
			multiErr = multierror.Append(multiErr, fmt.Errorf("cluster.id is required"))
		}

		if bundle.Cluster.Secret == "" {
			multiErr = multierror.Append(multiErr, fmt.Errorf("cluster.secret is required"))
		}
	}

	if bundle.Secrets == nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("secrets is required"))
	} else {
		if bundle.Secrets.BootstrapToken == "" {
			multiErr = multierror.Append(multiErr, fmt.Errorf("secrets.bootstraptoken is required"))
		}

		if bundle.Secrets.AESCBCEncryptionSecret == "" && bundle.Secrets.SecretboxEncryptionSecret == "" {
			multiErr = multierror.Append(multiErr, fmt.Errorf("one of [secrets.secretboxencryptionsecret, secrets.aescbcencryptionsecret] is required"))
		}

		if bundle.Secrets.AESCBCEncryptionSecret != "" && bundle.Secrets.SecretboxEncryptionSecret != "" {
			multiErr = multierror.Append(multiErr, fmt.Errorf("only one of [secrets.secretboxencryptionsecret, secrets.aescbcencryptionsecret] is allowed"))
		}
	}

	if bundle.TrustdInfo == nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("trustdinfo is required"))
	} else if bundle.TrustdInfo.Token == "" {
		multiErr = multierror.Append(multiErr, fmt.Errorf("trustdinfo.token is required"))
	}

	if err := bundle.validateCerts(); err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	return multiErr
}

//nolint:gocyclo,cyclop
func (bundle *Bundle) validateCerts() error {
	if bundle.Certs == nil {
		return errors.New("certs is required")
	}

	var multiErr error

	if bundle.Certs.OS == nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.os is required"))
	} else if err := validatePEMEncodedCertificateAndKey(bundle.Certs.OS); err != nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.os is invalid: %w", err))
	}

	return multiErr
}
