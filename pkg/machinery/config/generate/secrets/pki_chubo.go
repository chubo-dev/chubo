// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package secrets

import "github.com/chubo-dev/chubo/pkg/machinery/config"

// In the `chubo` build variant, Kubernetes/etcd PKI is intentionally omitted.
func (bundle *Bundle) populateKubernetesEtcd(_ *config.VersionContract) error {
	return nil
}

func (bundle *Bundle) validateKubernetesEtcdCerts() error {
	return nil
}
