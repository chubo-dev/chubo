// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cluster

import (
	"context"
	"fmt"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/gen/optional"
	"go.uber.org/zap"

	"github.com/chubo-dev/chubo/pkg/machinery/resources/cluster"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/network"
	"github.com/chubo-dev/chubo/pkg/machinery/version"
)

// LocalAffiliateController builds Affiliate resource for the local node.
type LocalAffiliateController struct{}

// Name implements controller.Controller interface.
func (ctrl *LocalAffiliateController) Name() string {
	return "cluster.LocalAffiliateController"
}

// Inputs implements controller.Controller interface.
func (ctrl *LocalAffiliateController) Inputs() []controller.Input {
	return []controller.Input{
		{
			Namespace: config.NamespaceName,
			Type:      cluster.ConfigType,
			ID:        optional.Some(cluster.ConfigID),
			Kind:      controller.InputWeak,
		},
		{
			Namespace: cluster.NamespaceName,
			Type:      cluster.IdentityType,
			ID:        optional.Some(cluster.LocalIdentity),
			Kind:      controller.InputWeak,
		},
		{
			Namespace: network.NamespaceName,
			Type:      network.HostnameStatusType,
			ID:        optional.Some(network.HostnameID),
			Kind:      controller.InputWeak,
		},
		{
			Namespace: network.NamespaceName,
			Type:      network.NodeAddressType,
			Kind:      controller.InputWeak,
		},
		{
			Namespace: config.NamespaceName,
			Type:      config.MachineTypeType,
			ID:        optional.Some(config.MachineTypeID),
			Kind:      controller.InputWeak,
		},
	}
}

// Outputs implements controller.Controller interface.
func (ctrl *LocalAffiliateController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: cluster.AffiliateType,
			Kind: controller.OutputShared,
		},
	}
}

// Run implements controller.Controller interface.
//
//nolint:gocyclo,cyclop
func (ctrl *LocalAffiliateController) Run(ctx context.Context, r controller.Runtime, _ *zap.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		}

		// mandatory resources to be fetched
		discoveryConfig, err := safe.ReaderGetByID[*cluster.Config](ctx, r, cluster.ConfigID)
		if err != nil {
			if !state.IsNotFoundError(err) {
				return fmt.Errorf("error getting discovery config: %w", err)
			}

			continue
		}

		identity, err := safe.ReaderGetByID[*cluster.Identity](ctx, r, cluster.LocalIdentity)
		if err != nil {
			if !state.IsNotFoundError(err) {
				return fmt.Errorf("error getting local identity: %w", err)
			}

			continue
		}

		hostname, err := safe.ReaderGetByID[*network.HostnameStatus](ctx, r, network.HostnameID)
		if err != nil {
			if !state.IsNotFoundError(err) {
				return fmt.Errorf("error getting hostname: %w", err)
			}

			continue
		}

		routedAddresses, err := safe.ReaderGetByID[*network.NodeAddress](ctx, r, network.NodeAddressRoutedID)
		if err != nil {
			if !state.IsNotFoundError(err) {
				return fmt.Errorf("error getting addresses: %w", err)
			}

			continue
		}

		machineType, err := safe.ReaderGetByID[*config.MachineType](ctx, r, config.MachineTypeID)
		if err != nil {
			if !state.IsNotFoundError(err) {
				return fmt.Errorf("error getting machine type: %w", err)
			}

			continue
		}

		localID := identity.TypedSpec().NodeID

		touchedIDs := map[resource.ID]struct{}{}

		if discoveryConfig.TypedSpec().DiscoveryEnabled {
			if err = safe.WriterModify(ctx, r, cluster.NewAffiliate(cluster.NamespaceName, localID), func(res *cluster.Affiliate) error {
				spec := res.TypedSpec()

				spec.NodeID = localID
				spec.Hostname = hostname.TypedSpec().FQDN()
				// In Chubo, there is no dedicated nodename resource. Use the machine FQDN
				// as the stable cluster node identifier.
				spec.Nodename = hostname.TypedSpec().FQDN()
				spec.MachineType = machineType.MachineType()
				spec.OperatingSystem = fmt.Sprintf("%s (%s)", version.Name, version.Tag)

				// Control plane metadata is legacy control-plane specific (API server port), so it is not
				// populated in Chubo.
				spec.ControlPlane = nil

				routedNodeIPs := routedAddresses.TypedSpec().IPs()

				spec.Addresses = routedNodeIPs

				return nil
			}); err != nil {
				return err
			}

			touchedIDs[localID] = struct{}{}
		}

		// list keys for cleanup
		list, err := safe.ReaderListAll[*cluster.Affiliate](ctx, r)
		if err != nil {
			return fmt.Errorf("error listing resources: %w", err)
		}

		for res := range list.All() {
			if res.Metadata().Owner() != ctrl.Name() {
				continue
			}

			if _, ok := touchedIDs[res.Metadata().ID()]; !ok {
				if err = r.Destroy(ctx, res.Metadata()); err != nil {
					return fmt.Errorf("error cleaning up specs: %w", err)
				}
			}
		}

		r.ResetRestartBackoff()
	}
}
