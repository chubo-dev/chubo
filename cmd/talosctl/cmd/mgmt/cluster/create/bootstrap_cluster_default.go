//go:build !chubo

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package create

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/cluster/create/clusterops"
	"github.com/siderolabs/talos/pkg/cluster/check"
	"github.com/siderolabs/talos/pkg/provision/access"
)

func bootstrapCluster(ctx context.Context, clusterAccess *access.Adapter, cOps clusterops.Common) error {
	if !cOps.WithInitNode {
		if err := clusterAccess.Bootstrap(ctx, os.Stdout); err != nil {
			return fmt.Errorf("bootstrap error: %w", err)
		}
	}

	if !cOps.ClusterWait {
		return nil
	}

	// Run cluster readiness checks.
	checkCtx, checkCtxCancel := context.WithTimeout(ctx, cOps.ClusterWaitTimeout)
	defer checkCtxCancel()

	checks := check.DefaultClusterChecks()

	if cOps.SkipK8sNodeReadinessCheck {
		checks = slices.Concat(check.PreBootSequenceChecks(), check.K8sComponentsReadinessChecks())
	}

	checks = slices.Concat(checks, check.ExtraClusterChecks())

	if err := check.Wait(checkCtx, clusterAccess, checks, check.StderrReporter()); err != nil {
		return err
	}

	if cOps.SkipKubeconfig {
		return nil
	}

	return mergeKubeconfig(ctx, clusterAccess)
}
