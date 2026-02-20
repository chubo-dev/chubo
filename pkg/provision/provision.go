// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package provision provides abstract definitions for Chubo cluster provisioners.
package provision

import (
	"context"

	"github.com/chubo-dev/chubo/pkg/machinery/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config/bundle"
	"github.com/chubo-dev/chubo/pkg/machinery/config/generate"
	"github.com/chubo-dev/chubo/pkg/machinery/config/types/v1alpha1"
)

// Provisioner is an interface each provisioner should implement.
//
//nolint:interfacebloat
type Provisioner interface {
	Create(context.Context, ClusterRequest, ...Option) (Cluster, error)
	Destroy(context.Context, Cluster, ...Option) error

	Reflect(ctx context.Context, clusterName, stateDirectory string) (Cluster, error)

	GenOptions(NetworkRequest, *config.VersionContract) ([]generate.Option, []bundle.Option)

	GetInClusterControlPlaneEndpoint(req NetworkRequest, controlPlanePort int) string
	GetExternalControlPlaneEndpoint(req NetworkRequest, controlPlanePort int) string
	// GetTalosAPIEndpoints is a legacy compatibility method name.
	GetTalosAPIEndpoints(NetworkRequest) []string

	GetFirstInterface() v1alpha1.IfaceSelector
	GetFirstInterfaceName() string

	Close() error

	UserDiskName(index int) string
}

type chuboAPIEndpointsProvider interface {
	GetChuboAPIEndpoints(NetworkRequest) []string
}

// GetChuboAPIEndpoints returns the OS API endpoints for a provisioner.
func GetChuboAPIEndpoints(p Provisioner, req NetworkRequest) []string {
	if provider, ok := any(p).(chuboAPIEndpointsProvider); ok {
		return provider.GetChuboAPIEndpoints(req)
	}

	return p.GetTalosAPIEndpoints(req)
}
