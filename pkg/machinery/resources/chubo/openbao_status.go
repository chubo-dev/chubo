// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/meta"
	"github.com/cosi-project/runtime/pkg/resource/protobuf"
	"github.com/cosi-project/runtime/pkg/resource/typed"

	"github.com/chubo-dev/chubo/pkg/machinery/proto"
)

// OpenBaoStatusType is the type of OpenBaoStatus resource.
const OpenBaoStatusType = resource.Type("OpenBaoStatuses.chubo.dev")

// OpenBaoStatusID is the single ID for the OpenBaoStatus resource.
const OpenBaoStatusID = resource.ID("openbao")

// OpenBaoStatus reports the state of the OS-managed OpenBao service.
type OpenBaoStatus = typed.Resource[OpenBaoStatusSpec, OpenBaoStatusExtension]

// OpenBaoStatusSpec describes OpenBao service status.
//
//gotagsrewrite:gen
type OpenBaoStatusSpec struct {
	// Configured indicates whether machine config requested host-native OpenBao.
	Configured bool `yaml:"configured" protobuf:"1"`
	// Mode is the requested OpenBao mode.
	Mode string `yaml:"mode,omitempty" protobuf:"2"`
	// Running reflects v1alpha1 service running state.
	Running bool `yaml:"running" protobuf:"3"`
	// Healthy reflects v1alpha1 service health state.
	Healthy bool `yaml:"healthy" protobuf:"4"`
	// Initialized reports whether OpenBao has been initialized.
	Initialized bool `yaml:"initialized" protobuf:"5"`
	// Sealed reports whether OpenBao is currently sealed.
	Sealed bool `yaml:"sealed" protobuf:"6"`
	// LastError captures the last probe/init/unseal error.
	LastError string `yaml:"lastError,omitempty" protobuf:"7"`
}

// DeepCopy generates a deep copy of OpenBaoStatusSpec.
func (o OpenBaoStatusSpec) DeepCopy() OpenBaoStatusSpec {
	return o
}

// NewOpenBaoStatus initializes an OpenBaoStatus resource.
func NewOpenBaoStatus() *OpenBaoStatus {
	return typed.NewResource[OpenBaoStatusSpec, OpenBaoStatusExtension](
		resource.NewMetadata(NamespaceName, OpenBaoStatusType, OpenBaoStatusID, resource.VersionUndefined),
		OpenBaoStatusSpec{},
	)
}

// OpenBaoStatusExtension provides auxiliary methods for OpenBaoStatus.
type OpenBaoStatusExtension struct{}

// ResourceDefinition implements [typed.Extension] interface.
func (OpenBaoStatusExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type: OpenBaoStatusType,
		Aliases: []resource.Type{
			"openbaostatus",
			"openbao",
		},
		DefaultNamespace: NamespaceName,
		PrintColumns: []meta.PrintColumn{
			{Name: "Configured", JSONPath: `{.configured}`},
			{Name: "Mode", JSONPath: `{.mode}`},
			{Name: "Running", JSONPath: `{.running}`},
			{Name: "Healthy", JSONPath: `{.healthy}`},
			{Name: "Initialized", JSONPath: `{.initialized}`},
			{Name: "Sealed", JSONPath: `{.sealed}`},
			{Name: "LastError", JSONPath: `{.lastError}`},
		},
	}
}

func init() {
	proto.RegisterDefaultTypes()

	err := protobuf.RegisterDynamic[OpenBaoStatusSpec](OpenBaoStatusType, &OpenBaoStatus{})
	if err != nil {
		panic(err)
	}
}
