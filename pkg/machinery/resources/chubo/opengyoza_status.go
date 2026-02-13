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

// OpenGyozaStatusType is the type of OpenGyozaStatus resource.
const OpenGyozaStatusType = resource.Type("OpenGyozaStatuses.chubo.dev")

// OpenGyozaStatusID is the single ID for the OpenGyozaStatus resource.
const OpenGyozaStatusID = resource.ID("opengyoza")

// OpenGyozaStatus reports the state of the OS-managed opengyoza service.
type OpenGyozaStatus = typed.Resource[OpenGyozaStatusSpec, OpenGyozaStatusExtension]

// OpenGyozaStatusSpec describes opengyoza status.
//
//gotagsrewrite:gen
type OpenGyozaStatusSpec struct {
	// Configured indicates whether machine config requested opengyoza.
	Configured bool `yaml:"configured" protobuf:"1"`
	// Role is the requested service role (server|client).
	Role string `yaml:"role,omitempty" protobuf:"2"`
	// Running reflects v1alpha1 service running state.
	Running bool `yaml:"running" protobuf:"3"`
	// Healthy reflects v1alpha1 service health state.
	Healthy bool `yaml:"healthy" protobuf:"4"`
	// BinaryMode reports whether the running binary is a fallback mock or an artifact.
	BinaryMode string `yaml:"binaryMode,omitempty" protobuf:"5"`
	// Leader is the observed opengyoza leader address (best-effort).
	Leader string `yaml:"leader,omitempty" protobuf:"6"`
	// PeerCount is the observed number of opengyoza peers (best-effort).
	PeerCount int32 `yaml:"peerCount,omitempty" protobuf:"7"`
	// LastError captures the last API query error (best-effort).
	LastError string `yaml:"lastError,omitempty" protobuf:"8"`
	// ACLReady indicates whether ACL-protected API calls succeed with the OS-derived token.
	ACLReady bool `yaml:"aclReady" protobuf:"9"`
	// ACLLastError captures the last ACL bootstrap/verification error (best-effort).
	ACLLastError string `yaml:"aclLastError,omitempty" protobuf:"10"`
}

// DeepCopy generates a deep copy of OpenGyozaStatusSpec.
func (o OpenGyozaStatusSpec) DeepCopy() OpenGyozaStatusSpec {
	return o
}

// NewOpenGyozaStatus initializes an OpenGyozaStatus resource.
func NewOpenGyozaStatus() *OpenGyozaStatus {
	return typed.NewResource[OpenGyozaStatusSpec, OpenGyozaStatusExtension](
		resource.NewMetadata(NamespaceName, OpenGyozaStatusType, OpenGyozaStatusID, resource.VersionUndefined),
		OpenGyozaStatusSpec{},
	)
}

// OpenGyozaStatusExtension provides auxiliary methods for OpenGyozaStatus.
type OpenGyozaStatusExtension struct{}

// ResourceDefinition implements [typed.Extension] interface.
func (OpenGyozaStatusExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type: OpenGyozaStatusType,
		Aliases: []resource.Type{
			"opengyozastatus",
			"opengyoza",
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
				Name:     "ACLReady",
				JSONPath: `{.aclReady}`,
			},
			{
				Name:     "Peers",
				JSONPath: `{.peerCount}`,
			},
			{
				Name:     "Leader",
				JSONPath: `{.leader}`,
			},
			{
				Name:     "BinaryMode",
				JSONPath: `{.binaryMode}`,
			},
			{
				Name:     "LastError",
				JSONPath: `{.lastError}`,
			},
		},
	}
}

func init() {
	proto.RegisterDefaultTypes()

	err := protobuf.RegisterDynamic[OpenGyozaStatusSpec](OpenGyozaStatusType, &OpenGyozaStatus{})
	if err != nil {
		panic(err)
	}
}
