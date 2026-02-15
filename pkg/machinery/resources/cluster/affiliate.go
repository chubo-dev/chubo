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

	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/proto"
)

//go:generate go tool github.com/siderolabs/deep-copy -type AffiliateSpec -type ConfigSpec -type ControlPlaneEndpointSpec -type IdentitySpec -type MemberSpec -type InfoSpec -header-file ../../../../hack/boilerplate.txt -o deep_copy.generated.go .

// AffiliateType is type of Affiliate resource.
const AffiliateType = resource.Type("Affiliates.cluster.talos.dev")

// Affiliate resource holds information about a cluster affiliate: it is a discovered potential cluster member.
//
// Controller builds local Affiliate structure for the node itself, other Affiliates are pulled from the registry during the discovery process.
type Affiliate = typed.Resource[AffiliateSpec, AffiliateExtension]

// NewAffiliate initializes the Affiliate resource.
func NewAffiliate(namespace resource.Namespace, id resource.ID) *Affiliate {
	return typed.NewResource[AffiliateSpec, AffiliateExtension](
		resource.NewMetadata(namespace, AffiliateType, id, resource.VersionUndefined),
		AffiliateSpec{},
	)
}

// AffiliateExtension provides auxiliary methods for Affiliate.
type AffiliateExtension struct{}

// ResourceDefinition implements [typed.Extension] interface.
func (r AffiliateExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type:             AffiliateType,
		Aliases:          []resource.Type{},
		DefaultNamespace: NamespaceName,
		PrintColumns: []meta.PrintColumn{
			{
				Name:     "Hostname",
				JSONPath: `{.hostname}`,
			},
			{
				Name:     "Machine Type",
				JSONPath: `{.machineType}`,
			},
			{
				Name:     "Addresses",
				JSONPath: `{.addresses}`,
			},
		},
	}
}

// AffiliateSpec describes Affiliate state.
//
//gotagsrewrite:gen
type AffiliateSpec struct {
	NodeID          string        `yaml:"nodeId" protobuf:"1"`
	Addresses       []netip.Addr  `yaml:"addresses" protobuf:"2"`
	Hostname        string        `yaml:"hostname" protobuf:"3"`
	Nodename        string        `yaml:"nodename,omitempty" protobuf:"4"`
	OperatingSystem string        `yaml:"operatingSystem" protobuf:"5"`
	MachineType     machine.Type  `yaml:"machineType" protobuf:"6"`
	ControlPlane    *ControlPlane `yaml:"controlPlane,omitempty" protobuf:"8"`
}

// ControlPlane describes ControlPlane data if any.
//
//gotagsrewrite:gen
type ControlPlane struct {
	APIServerPort int `yaml:"port" protobuf:"1"`
}

// Merge two AffiliateSpecs.
//
//nolint:gocyclo
func (spec *AffiliateSpec) Merge(other *AffiliateSpec) {
	for _, addr := range other.Addresses {
		found := slices.Contains(spec.Addresses, addr)

		if !found {
			spec.Addresses = append(spec.Addresses, addr)
		}
	}

	if other.ControlPlane != nil {
		spec.ControlPlane = other.ControlPlane
	}

	if other.Hostname != "" {
		spec.Hostname = other.Hostname
	}

	if other.Nodename != "" {
		spec.Nodename = other.Nodename
	}

	if other.MachineType != machine.TypeUnknown {
		spec.MachineType = other.MachineType
	}
}

func init() {
	proto.RegisterDefaultTypes()

	err := protobuf.RegisterDynamic[AffiliateSpec](AffiliateType, &Affiliate{})
	if err != nil {
		panic(err)
	}
}
