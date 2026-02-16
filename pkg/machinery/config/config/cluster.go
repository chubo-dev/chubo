// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

// ClusterConfig defines the requirements for a config that pertains to
// cluster-related options.
//
// In the Chubo fork, this intentionally excludes legacy control-plane concepts and
// keeps only cluster identity + discovery inputs used by the OS API surface.
type ClusterConfig interface {
	ID() string
	Name() string
	Secret() string
	Discovery() Discovery
}

// Discovery describes cluster membership discovery.
type Discovery interface {
	Enabled() bool
	Service() ServiceRegistry
}

// ServiceRegistry describes external service discovery registry.
type ServiceRegistry interface {
	Enabled() bool
	Endpoint() string
}
