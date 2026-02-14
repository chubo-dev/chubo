// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package operator

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/chubo-dev/chubo/pkg/machinery/resources/network"
	"github.com/cosi-project/runtime/pkg/state"
)

// VIP implements a no-op Virtual IP operator for chubo mode.
type VIP struct {
	logger   *zap.Logger
	linkName string
}

// NewVIP creates Virtual IP operator.
func NewVIP(logger *zap.Logger, linkName string, _ network.VIPOperatorSpec, _ state.State) *VIP {
	return &VIP{
		logger:   logger,
		linkName: linkName,
	}
}

// Prefix returns unique operator prefix which gets prepended to each spec.
func (vip *VIP) Prefix() string {
	return fmt.Sprintf("vip/%s", vip.linkName)
}

// Run the operator loop.
func (vip *VIP) Run(ctx context.Context, _ chan<- struct{}) {
	vip.logger.Warn("VIP operator is disabled in chubo mode", zap.String("link", vip.linkName))
	<-ctx.Done()
}

// AddressSpecs implements Operator interface.
func (vip *VIP) AddressSpecs() []network.AddressSpecSpec {
	return nil
}

// LinkSpecs implements Operator interface.
func (vip *VIP) LinkSpecs() []network.LinkSpecSpec {
	return nil
}

// RouteSpecs implements Operator interface.
func (vip *VIP) RouteSpecs() []network.RouteSpecSpec {
	return nil
}

// HostnameSpecs implements Operator interface.
func (vip *VIP) HostnameSpecs() []network.HostnameSpecSpec {
	return nil
}

// ResolverSpecs implements Operator interface.
func (vip *VIP) ResolverSpecs() []network.ResolverSpecSpec {
	return nil
}

// TimeServerSpecs implements Operator interface.
func (vip *VIP) TimeServerSpecs() []network.TimeServerSpecSpec {
	return nil
}
