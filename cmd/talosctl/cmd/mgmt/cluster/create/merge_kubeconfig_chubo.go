//go:build chubo

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package create

import (
	"context"
	"errors"

	"github.com/chubo-dev/chubo/pkg/provision/access"
)

func mergeKubeconfig(ctx context.Context, clusterAccess *access.Adapter) error {
	_ = ctx
	_ = clusterAccess

	return errors.New("kubeconfig export is not supported in chubo build (use --skip-kubeconfig)")
}
