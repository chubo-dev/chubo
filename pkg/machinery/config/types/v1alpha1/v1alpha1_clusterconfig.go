// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha1

import "github.com/chubo-dev/chubo/pkg/machinery/config/config"

// Verify interface.
var _ config.ClusterConfig = (*ClusterConfig)(nil)

// ID returns the unique identifier for the cluster.
func (c *ClusterConfig) ID() string {
	if c == nil {
		return ""
	}

	return c.ClusterID
}

// Name returns the configured cluster name.
func (c *ClusterConfig) Name() string {
	if c == nil {
		return ""
	}

	return c.ClusterName
}

// Secret returns the cluster secret used by discovery/service membership.
func (c *ClusterConfig) Secret() string {
	if c == nil {
		return ""
	}

	return c.ClusterSecret
}

// Discovery returns cluster membership discovery configuration.
func (c *ClusterConfig) Discovery() config.Discovery {
	if c == nil || c.ClusterDiscoveryConfig == nil {
		return &ClusterDiscoveryConfig{}
	}

	return c.ClusterDiscoveryConfig
}
