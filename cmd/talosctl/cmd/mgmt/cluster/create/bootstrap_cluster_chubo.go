//go:build chubo

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package create

import (
	"context"
	"fmt"
	"os"

	"github.com/chubo-dev/chubo/cmd/talosctl/cmd/mgmt/cluster/create/clusterops"
	"github.com/chubo-dev/chubo/pkg/provision/access"
)

func bootstrapCluster(ctx context.Context, clusterAccess *access.Adapter, cOps clusterops.Common) error {
	if !cOps.WithInitNode {
		if err := clusterAccess.Bootstrap(ctx, os.Stdout); err != nil {
			return fmt.Errorf("bootstrap error: %w", err)
		}
	}

	if cOps.ClusterWait {
		fmt.Fprintln(os.Stderr, "chubo build: skipping Kubernetes/etcd readiness checks for `cluster create`.")
	}

	if cOps.SkipKubeconfig {
		return nil
	}

	return mergeKubeconfig(ctx, clusterAccess)
}
