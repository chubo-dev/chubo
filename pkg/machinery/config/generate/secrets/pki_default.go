// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo && !chuboos

package secrets

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/siderolabs/crypto/x509"

	"github.com/siderolabs/talos/pkg/machinery/config"
)

// populateKubernetesEtcd fills Kubernetes/etcd PKI into the secrets bundle.
//
// This is isolated behind build tags so the `chubo` build variant can omit
// Kubernetes/etcd PKI entirely.
func (bundle *Bundle) populateKubernetesEtcd(versionContract *config.VersionContract) error {
	if bundle.Certs.Etcd == nil {
		etcd, err := NewEtcdCA(bundle.Clock.Now(), versionContract)
		if err != nil {
			return err
		}

		bundle.Certs.Etcd = &x509.PEMEncodedCertificateAndKey{
			Crt: etcd.CrtPEM,
			Key: etcd.KeyPEM,
		}
	}

	if bundle.Certs.K8s == nil {
		kubernetesCA, err := NewKubernetesCA(bundle.Clock.Now(), versionContract)
		if err != nil {
			return err
		}

		bundle.Certs.K8s = &x509.PEMEncodedCertificateAndKey{
			Crt: kubernetesCA.CrtPEM,
			Key: kubernetesCA.KeyPEM,
		}
	}

	if bundle.Certs.K8sAggregator == nil {
		aggregatorCA, err := NewAggregatorCA(bundle.Clock.Now(), versionContract)
		if err != nil {
			return err
		}

		bundle.Certs.K8sAggregator = &x509.PEMEncodedCertificateAndKey{
			Crt: aggregatorCA.CrtPEM,
			Key: aggregatorCA.KeyPEM,
		}
	}

	if bundle.Certs.K8sServiceAccount == nil {
		if versionContract.UseRSAServiceAccountKey() {
			serviceAccount, err := x509.NewRSAKey()
			if err != nil {
				return err
			}

			bundle.Certs.K8sServiceAccount = &x509.PEMEncodedKey{
				Key: serviceAccount.KeyPEM,
			}
		} else {
			serviceAccount, err := x509.NewECDSAKey()
			if err != nil {
				return err
			}

			bundle.Certs.K8sServiceAccount = &x509.PEMEncodedKey{
				Key: serviceAccount.KeyPEM,
			}
		}
	}

	return nil
}

func (bundle *Bundle) validateKubernetesEtcdCerts() error {
	var multiErr error

	if bundle.Certs.Etcd == nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.etcd is required"))
	} else if err := validatePEMEncodedCertificateAndKey(bundle.Certs.Etcd); err != nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.etcd is invalid: %w", err))
	}

	if bundle.Certs.K8s == nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.k8s is required"))
	} else if err := validatePEMEncodedCertificateAndKey(bundle.Certs.K8s); err != nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.k8s is invalid: %w", err))
	}

	if bundle.Certs.K8sAggregator == nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.k8saggregator is required"))
	} else if err := validatePEMEncodedCertificateAndKey(bundle.Certs.K8sAggregator); err != nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.k8saggregator is invalid: %w", err))
	}

	if bundle.Certs.K8sServiceAccount == nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.k8sserviceaccount is required"))
	} else if _, err := bundle.Certs.K8sServiceAccount.GetKey(); err != nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("certs.k8sserviceaccount.key is invalid: %w", err))
	}

	return multiErr
}
