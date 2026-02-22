// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha1

import (
	"github.com/siderolabs/go-pointer"

	"github.com/chubo-dev/chubo/pkg/machinery/config/config"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

// Enabled implements the config.Discovery interface.
func (c *ClusterDiscoveryConfig) Enabled() bool {
	return pointer.SafeDeref(c.DiscoveryEnabled)
}

// Service implements the config.Discovery interface.
func (c *ClusterDiscoveryConfig) Service() config.ServiceRegistry {
	return c.DiscoveryRegistries.RegistryService
}

// Enabled implements the config.ServiceRegistry interface.
func (c RegistryServiceConfig) Enabled() bool {
	return !pointer.SafeDeref(c.RegistryDisabled)
}

// Endpoint implements the config.ServiceRegistry interface.
func (c RegistryServiceConfig) Endpoint() string {
	if c.RegistryEndpoint == "" {
		return constants.EffectiveDiscoveryServiceEndpoint()
	}

	return c.RegistryEndpoint
}
