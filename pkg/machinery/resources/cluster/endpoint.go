// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cluster

import (
	"net/netip"
	"slices"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/meta"
	"github.com/cosi-project/runtime/pkg/resource/protobuf"
	"github.com/cosi-project/runtime/pkg/resource/typed"
	"github.com/siderolabs/gen/xslices"

	"github.com/chubo-dev/chubo/pkg/machinery/proto"
)

// ControlPlaneNamespaceName contains resources used to reach control plane services (trustd/apid).
const ControlPlaneNamespaceName resource.Namespace = "controlplane"

// ControlPlaneEndpointType is type of ControlPlaneEndpoint resource.
const ControlPlaneEndpointType = resource.Type("ControlPlaneEndpoints.cluster.talos.dev")

// ControlPlaneDiscoveredEndpointsID is the resource ID for discovery-derived endpoints.
const ControlPlaneDiscoveredEndpointsID = resource.ID("discovery")

// ControlPlaneEndpoint resource holds control-plane endpoints which workers can use to reach trustd/apid.
type ControlPlaneEndpoint = typed.Resource[ControlPlaneEndpointSpec, ControlPlaneEndpointExtension]

// ControlPlaneEndpointSpec describes a list of endpoints to connect to.
//
//gotagsrewrite:gen
type ControlPlaneEndpointSpec struct {
	Addresses []netip.Addr `yaml:"addresses" protobuf:"1"`
	Hosts     []string     `yaml:"hosts" protobuf:"2"`
}

// NewControlPlaneEndpoint initializes the ControlPlaneEndpoint resource.
func NewControlPlaneEndpoint(namespace resource.Namespace, id resource.ID) *ControlPlaneEndpoint {
	return typed.NewResource[ControlPlaneEndpointSpec, ControlPlaneEndpointExtension](
		resource.NewMetadata(namespace, ControlPlaneEndpointType, id, resource.VersionUndefined),
		ControlPlaneEndpointSpec{},
	)
}

// ControlPlaneEndpointExtension provides auxiliary methods for ControlPlaneEndpoint.
type ControlPlaneEndpointExtension struct{}

// ResourceDefinition implements [typed.Extension] interface.
func (ControlPlaneEndpointExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type:             ControlPlaneEndpointType,
		Aliases:          []resource.Type{},
		DefaultNamespace: ControlPlaneNamespaceName,
		PrintColumns: []meta.PrintColumn{
			{
				Name:     "Addresses",
				JSONPath: "{.addresses}",
			},
			{
				Name:     "Hosts",
				JSONPath: "{.hosts}",
			},
		},
	}
}

// ControlPlaneEndpointList is a flattened list of endpoints.
type ControlPlaneEndpointList struct {
	Addresses []netip.Addr
	Hosts     []string
}

// Merge endpoints from multiple ControlPlaneEndpoint resources into a single list.
func (l ControlPlaneEndpointList) Merge(endpoint *ControlPlaneEndpoint) ControlPlaneEndpointList {
	for _, ip := range endpoint.TypedSpec().Addresses {
		idx, _ := slices.BinarySearchFunc(l.Addresses, ip, func(a netip.Addr, target netip.Addr) int {
			return a.Compare(target)
		})
		if idx < len(l.Addresses) && l.Addresses[idx].Compare(ip) == 0 {
			continue
		}

		l.Addresses = slices.Insert(l.Addresses, idx, ip)
	}

	for _, host := range endpoint.TypedSpec().Hosts {
		idx, _ := slices.BinarySearch(l.Hosts, host)
		if idx < len(l.Hosts) && l.Hosts[idx] == host {
			continue
		}

		l.Hosts = slices.Insert(l.Hosts, idx, host)
	}

	return l
}

// IsEmpty checks if the ControlPlaneEndpointList is empty.
func (l ControlPlaneEndpointList) IsEmpty() bool {
	return len(l.Addresses) == 0 && len(l.Hosts) == 0
}

// Strings returns a slice of formatted endpoints to string.
func (l ControlPlaneEndpointList) Strings() []string {
	return slices.Concat(
		xslices.Map(l.Addresses, netip.Addr.String),
		l.Hosts,
	)
}

func init() {
	proto.RegisterDefaultTypes()

	err := protobuf.RegisterDynamic[ControlPlaneEndpointSpec](ControlPlaneEndpointType, &ControlPlaneEndpoint{})
	if err != nil {
		panic(err)
	}
}
