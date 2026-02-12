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

// BootstrapStatusType is the type of BootstrapStatus resource.
const BootstrapStatusType = resource.Type("BootstrapStatuses.chubo.dev")

// BootstrapStatusID is the single ID for the BootstrapStatus resource.
const BootstrapStatusID = resource.ID("bootstrap")

// BootstrapStatus reports the state of the Chubo bootstrap payload rendering.
type BootstrapStatus = typed.Resource[BootstrapStatusSpec, BootstrapStatusExtension]

// BootstrapStatusSpec describes bootstrap payload status.
//
//gotagsrewrite:gen
type BootstrapStatusSpec struct {
	// Configured indicates that the applied machine configuration requested a bootstrap payload.
	Configured bool `yaml:"configured" protobuf:"1"`
	// Rendered indicates that the verified payload was rendered to disk.
	Rendered bool `yaml:"rendered" protobuf:"2"`
	// SignerSha256 is the SHA-256 fingerprint of the signer certificate (informational).
	SignerSha256 string `yaml:"signerSha256,omitempty" protobuf:"3"`
}

// DeepCopy generates a deep copy of BootstrapStatusSpec.
func (o BootstrapStatusSpec) DeepCopy() BootstrapStatusSpec {
	return o
}

// NewBootstrapStatus initializes a BootstrapStatus resource.
func NewBootstrapStatus() *BootstrapStatus {
	return typed.NewResource[BootstrapStatusSpec, BootstrapStatusExtension](
		resource.NewMetadata(NamespaceName, BootstrapStatusType, BootstrapStatusID, resource.VersionUndefined),
		BootstrapStatusSpec{},
	)
}

// BootstrapStatusExtension provides auxiliary methods for BootstrapStatus.
type BootstrapStatusExtension struct{}

// ResourceDefinition implements [typed.Extension] interface.
func (BootstrapStatusExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type: BootstrapStatusType,
		Aliases: []resource.Type{
			"chubobootstrapstatus",
			"chubobootstrap",
		},
		DefaultNamespace: NamespaceName,
		PrintColumns: []meta.PrintColumn{
			{
				Name:     "Configured",
				JSONPath: `{.configured}`,
			},
			{
				Name:     "Rendered",
				JSONPath: `{.rendered}`,
			},
			{
				Name:     "SignerSha256",
				JSONPath: `{.signerSha256}`,
			},
		},
	}
}

func init() {
	proto.RegisterDefaultTypes()

	err := protobuf.RegisterDynamic[BootstrapStatusSpec](BootstrapStatusType, &BootstrapStatus{})
	if err != nil {
		panic(err)
	}
}
