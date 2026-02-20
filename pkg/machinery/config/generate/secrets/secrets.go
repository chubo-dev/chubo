// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package secrets provides types and methods to handle base machine configuration secrets.
package secrets

import (
	"time"

	"github.com/siderolabs/crypto/x509"
)

// CAValidityTime is the default validity time for CA certificates.
const CAValidityTime = 87600 * time.Hour

// Bundle contains all cluster secrets required to generate machine configuration.
//
// NB: this structure is marshalled/unmarshalled to/from JSON in various projects, so
// we need to keep representation compatible.
type Bundle struct {
	Clock      Clock       `yaml:"-" json:"-"`
	Cluster    *Cluster    `json:"Cluster"`
	Secrets    *Secrets    `json:"Secrets"`
	TrustdInfo *TrustdInfo `json:"TrustdInfo"`
	Certs      *Certs      `json:"Certs"`
}

// Certs holds the base64 encoded keys and certificates.
type Certs struct {
	// Store is the legacy datastore CA certificate and key.
	Store *x509.PEMEncodedCertificateAndKey `json:"Store"`
	// Workload is the legacy workload CA certificate and key.
	Workload *x509.PEMEncodedCertificateAndKey `json:"Workload"`
	// WorkloadProxy is the legacy workload proxy CA certificate and key.
	WorkloadProxy *x509.PEMEncodedCertificateAndKey `json:"WorkloadProxy"`
	// WorkloadSigner is the legacy workload service account signing key.
	WorkloadSigner *x509.PEMEncodedKey `json:"WorkloadSigner"`
	// OS is the OS API CA certificate and key.
	OS *x509.PEMEncodedCertificateAndKey `json:"OS"`
}

// Cluster holds cluster-wide secrets.
type Cluster struct {
	ID     string `json:"Id"`
	Secret string `json:"Secret"`
}

// Secrets holds sensitive bootstrap data.
type Secrets struct {
	BootstrapToken            string `json:"BootstrapToken"`
	AESCBCEncryptionSecret    string `json:"AESCBCEncryptionSecret,omitempty" yaml:",omitempty"`
	SecretboxEncryptionSecret string `json:"SecretboxEncryptionSecret,omitempty" yaml:",omitempty"`
}

// TrustdInfo holds the trustd credentials.
type TrustdInfo struct {
	Token string `json:"Token"`
}
