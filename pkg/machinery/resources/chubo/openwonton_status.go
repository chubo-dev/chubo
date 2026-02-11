// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/meta"
	"github.com/cosi-project/runtime/pkg/resource/protobuf"
	"github.com/cosi-project/runtime/pkg/resource/typed"

	"github.com/siderolabs/talos/pkg/machinery/proto"
)

// OpenWontonStatusType is the type of OpenWontonStatus resource.
const OpenWontonStatusType = resource.Type("OpenWontonStatuses.chubo.dev")

// OpenWontonStatusID is the single ID for the OpenWontonStatus resource.
const OpenWontonStatusID = resource.ID("openwonton")

// OpenWontonStatus reports the state of the OS-managed openwonton service.
type OpenWontonStatus = typed.Resource[OpenWontonStatusSpec, OpenWontonStatusExtension]

// OpenWontonStatusSpec describes openwonton status.
//
//gotagsrewrite:gen
type OpenWontonStatusSpec struct {
	// Configured indicates whether machine config requested openwonton.
	Configured bool `yaml:"configured" protobuf:"1"`
	// Role is the requested service role (server|client).
	Role string `yaml:"role,omitempty" protobuf:"2"`
	// Running reflects v1alpha1 service running state.
	Running bool `yaml:"running" protobuf:"3"`
	// Healthy reflects v1alpha1 service health state.
	Healthy bool `yaml:"healthy" protobuf:"4"`
	// BinaryMode reports whether the running binary is a fallback mock or an artifact.
	BinaryMode string `yaml:"binaryMode,omitempty" protobuf:"5"`
}

// DeepCopy generates a deep copy of OpenWontonStatusSpec.
func (o OpenWontonStatusSpec) DeepCopy() OpenWontonStatusSpec {
	return o
}

// NewOpenWontonStatus initializes an OpenWontonStatus resource.
func NewOpenWontonStatus() *OpenWontonStatus {
	return typed.NewResource[OpenWontonStatusSpec, OpenWontonStatusExtension](
		resource.NewMetadata(NamespaceName, OpenWontonStatusType, OpenWontonStatusID, resource.VersionUndefined),
		OpenWontonStatusSpec{},
	)
}

// OpenWontonStatusExtension provides auxiliary methods for OpenWontonStatus.
type OpenWontonStatusExtension struct{}

// ResourceDefinition implements [typed.Extension] interface.
func (OpenWontonStatusExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type: OpenWontonStatusType,
		Aliases: []resource.Type{
			"openwontonstatus",
			"openwonton",
		},
		DefaultNamespace: NamespaceName,
		PrintColumns: []meta.PrintColumn{
			{
				Name:     "Configured",
				JSONPath: `{.configured}`,
			},
			{
				Name:     "Role",
				JSONPath: `{.role}`,
			},
			{
				Name:     "Running",
				JSONPath: `{.running}`,
			},
			{
				Name:     "Healthy",
				JSONPath: `{.healthy}`,
			},
			{
				Name:     "BinaryMode",
				JSONPath: `{.binaryMode}`,
			},
		},
	}
}

func init() {
	proto.RegisterDefaultTypes()

	err := protobuf.RegisterDynamic[OpenWontonStatusSpec](OpenWontonStatusType, &OpenWontonStatus{})
	if err != nil {
		panic(err)
	}
}
