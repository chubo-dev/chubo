//go:build chubo

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cluster

import (
	"context"
	"fmt"
	"io"

	"github.com/siderolabs/talos/pkg/provision"
)

// Crashdump is a no-op in chubo builds.
//
// Chubo support collection is handled by dedicated talosctl support flows.
func Crashdump(ctx context.Context, cluster provision.Cluster, logWriter io.Writer, zipFilePath string) {
	_ = ctx
	_ = cluster
	_ = zipFilePath

	fmt.Fprintln(logWriter, "chubo build: cluster crashdump helper is disabled")
}
