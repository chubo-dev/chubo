// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package check

import (
	"context"
	"slices"
	"time"

	"github.com/chubo-dev/chubo/pkg/conditions"
)

// DefaultClusterChecks returns a set of default Talos cluster readiness checks.
func DefaultClusterChecks() []ClusterCheck {
	return slices.Concat(
		PreBootSequenceChecks(),
		ExtraClusterChecks(),
	)
}

// K8sComponentsReadinessChecks returns Kubernetes component readiness checks.
//
// Chubo doesn't manage Kubernetes, so this is kept as a no-op compatibility hook.
func K8sComponentsReadinessChecks() []ClusterCheck {
	return nil
}

// ExtraClusterChecks returns a set of additional Talos cluster readiness checks which work only for newer versions of Talos.
//
// ExtraClusterChecks can't be used reliably in upgrade tests, as older versions might not pass the checks.
func ExtraClusterChecks() []ClusterCheck {
	return []ClusterCheck{}
}

// PreBootSequenceChecks returns a set of Talos cluster readiness checks which are run before boot sequence.
func PreBootSequenceChecks() []ClusterCheck {
	return []ClusterCheck{
		// wait for apid to be ready on all the nodes
		func(cluster ClusterInfo) conditions.Condition {
			return conditions.PollingCondition("apid to be ready", func(ctx context.Context) error {
				return ApidReadyAssertion(ctx, cluster)
			}, 5*time.Second)
		},

		// wait for all nodes to report their memory size
		func(cluster ClusterInfo) conditions.Condition {
			return conditions.PollingCondition("all nodes memory sizes", func(ctx context.Context) error {
				return AllNodesMemorySizes(ctx, cluster)
			}, 5*time.Second)
		},

		// wait for all nodes to report their disk size
		func(cluster ClusterInfo) conditions.Condition {
			return conditions.PollingCondition("all nodes disk sizes", func(ctx context.Context) error {
				return AllNodesDiskSizes(ctx, cluster)
			}, 5*time.Second)
		},

		// check diagnostics
		func(cluster ClusterInfo) conditions.Condition {
			return conditions.PollingCondition("no diagnostics", func(ctx context.Context) error {
				return NoDiagnostics(ctx, cluster)
			}, 5*time.Second)
		},

		// wait for all nodes to finish booting
		func(cluster ClusterInfo) conditions.Condition {
			return conditions.PollingCondition("all nodes to finish boot sequence", func(ctx context.Context) error {
				return AllNodesBootedAssertion(ctx, cluster)
			}, 5*time.Second)
		},
	}
}
