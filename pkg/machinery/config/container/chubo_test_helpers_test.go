// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo || chuboos

package container_test

import (
	"github.com/siderolabs/crypto/x509"

	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
)

// applyChuboOSValidationDefaults mutates cfg so that it passes the minimal
// `chubo` v1alpha1 validation path (OS API + trustd requirements).
func applyChuboOSValidationDefaults(cfg *v1alpha1.Config) {
	if cfg.MachineConfig == nil {
		cfg.MachineConfig = &v1alpha1.MachineConfig{}
	}

	cfg.MachineConfig.MachineType = "controlplane"
	cfg.MachineConfig.MachineToken = "token"
	cfg.MachineConfig.MachineCA = &x509.PEMEncodedCertificateAndKey{
		Crt: []byte("cert"),
		Key: []byte("key"),
	}
}
