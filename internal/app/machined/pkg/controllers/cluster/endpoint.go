// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cluster

import (
	"context"
	"fmt"
	"net/netip"
	"slices"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/safe"
	"go.uber.org/zap"

	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	clusterres "github.com/chubo-dev/chubo/pkg/machinery/resources/cluster"
)

// EndpointController looks up control plane endpoints.
type EndpointController struct{}

// Name implements controller.Controller interface.
func (ctrl *EndpointController) Name() string {
	return "cluster.EndpointController"
}

// Inputs implements controller.Controller interface.
func (ctrl *EndpointController) Inputs() []controller.Input {
	return []controller.Input{
		{
			Namespace: clusterres.NamespaceName,
			Type:      clusterres.MemberType,
			Kind:      controller.InputWeak,
		},
	}
}

// Outputs implements controller.Controller interface.
func (ctrl *EndpointController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: clusterres.ControlPlaneEndpointType,
			Kind: controller.OutputShared,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *EndpointController) Run(ctx context.Context, r controller.Runtime, logger *zap.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		}

		memberList, err := safe.ReaderListAll[*clusterres.Member](ctx, r)
		if err != nil {
			return fmt.Errorf("error listing members: %w", err)
		}

		var endpoints []netip.Addr

		for member := range memberList.All() {
			memberSpec := member.TypedSpec()

			if !(memberSpec.MachineType == machine.TypeControlPlane || memberSpec.MachineType == machine.TypeInit) {
				continue
			}

			endpoints = append(endpoints, memberSpec.Addresses...)
		}

		slices.SortFunc(endpoints, func(a, b netip.Addr) int { return a.Compare(b) })

		if err := safe.WriterModify(
			ctx,
			r,
			clusterres.NewControlPlaneEndpoint(clusterres.ControlPlaneNamespaceName, clusterres.ControlPlaneDiscoveredEndpointsID),
			func(r *clusterres.ControlPlaneEndpoint) error {
				if !slices.Equal(r.TypedSpec().Addresses, endpoints) {
					logger.Debug("updated controlplane endpoints", zap.Any("endpoints", endpoints))
				}

				r.TypedSpec().Addresses = endpoints

				return nil
			},
		); err != nil {
			return fmt.Errorf("error updating endpoints: %w", err)
		}

		r.ResetRestartBackoff()
	}
}
