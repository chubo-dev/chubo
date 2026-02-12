// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build linux || darwin

package providers

import (
	"context"

	"github.com/chubo-dev/chubo/pkg/provision"
	"github.com/chubo-dev/chubo/pkg/provision/providers/qemu"
)

func newQemu(ctx context.Context) (provision.Provisioner, error) {
	return qemu.NewProvisioner(ctx)
}
