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

// OpenGyozaBootstrapStatusType is the type of OpenGyozaBootstrapStatus resource.
const OpenGyozaBootstrapStatusType = resource.Type("OpenGyozaBootstrapStatuses.chubo.dev")

// OpenGyozaBootstrapStatusID is the single ID for the OpenGyozaBootstrapStatus resource.
const OpenGyozaBootstrapStatusID = resource.ID("opengyoza")

// OpenGyozaBootstrapStatus reports the bootstrap and cluster readiness of the OS-managed opengyoza service.
type OpenGyozaBootstrapStatus = typed.Resource[OpenGyozaBootstrapStatusSpec, OpenGyozaBootstrapStatusExtension]

// OpenGyozaBootstrapStatusSpec describes opengyoza bootstrap status.
//
//gotagsrewrite:gen
type OpenGyozaBootstrapStatusSpec struct {
	// Configured indicates whether machine config requested opengyoza.
	Configured bool `yaml:"configured" protobuf:"1"`
	// Role is the requested service role (server|client).
	Role string `yaml:"role,omitempty" protobuf:"2"`
	// BootstrapExpect is the desired peer count for server bootstrap/quorum.
	BootstrapExpect int32 `yaml:"bootstrapExpect,omitempty" protobuf:"3"`
	// Join is the desired join list for retry-join operations.
	Join []string `yaml:"join,omitempty" protobuf:"4"`
	// Running reflects v1alpha1 service running state.
	Running bool `yaml:"running" protobuf:"5"`
	// Healthy reflects v1alpha1 service health state.
	Healthy bool `yaml:"healthy" protobuf:"6"`
	// ACLReady indicates whether ACL-protected API calls succeed with the OS-derived token.
	ACLReady bool `yaml:"aclReady" protobuf:"7"`
	// ACLLastError captures the last ACL bootstrap/verification error (best-effort).
	ACLLastError string `yaml:"aclLastError,omitempty" protobuf:"8"`
	// Leader is the observed opengyoza leader address (best-effort).
	Leader string `yaml:"leader,omitempty" protobuf:"9"`
	// PeerCount is the observed number of opengyoza peers (best-effort).
	PeerCount int32 `yaml:"peerCount,omitempty" protobuf:"10"`
	// ClusterReady indicates whether the service appears bootstrapped and joined (best-effort).
	ClusterReady bool `yaml:"clusterReady" protobuf:"11"`
	// LastError captures the last bootstrap/cluster query error (best-effort).
	LastError string `yaml:"lastError,omitempty" protobuf:"12"`
	// ACLTokenSHA256 is a stable hash of the OS-derived ACL token (no secret material).
	ACLTokenSHA256 string `yaml:"aclTokenSha256,omitempty" protobuf:"13"`
}

// DeepCopy generates a deep copy of OpenGyozaBootstrapStatusSpec.
func (o OpenGyozaBootstrapStatusSpec) DeepCopy() OpenGyozaBootstrapStatusSpec {
	if len(o.Join) > 0 {
		cp := make([]string, len(o.Join))
		copy(cp, o.Join)
		o.Join = cp
	}

	return o
}

// NewOpenGyozaBootstrapStatus initializes an OpenGyozaBootstrapStatus resource.
func NewOpenGyozaBootstrapStatus() *OpenGyozaBootstrapStatus {
	return typed.NewResource[OpenGyozaBootstrapStatusSpec, OpenGyozaBootstrapStatusExtension](
		resource.NewMetadata(NamespaceName, OpenGyozaBootstrapStatusType, OpenGyozaBootstrapStatusID, resource.VersionUndefined),
		OpenGyozaBootstrapStatusSpec{},
	)
}

// OpenGyozaBootstrapStatusExtension provides auxiliary methods for OpenGyozaBootstrapStatus.
type OpenGyozaBootstrapStatusExtension struct{}

// ResourceDefinition implements [typed.Extension] interface.
func (OpenGyozaBootstrapStatusExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type: OpenGyozaBootstrapStatusType,
		Aliases: []resource.Type{
			"opengyozabootstrapstatus",
			"opengyozabootstrap",
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
				Name:     "ClusterReady",
				JSONPath: `{.clusterReady}`,
			},
			{
				Name:     "Peers",
				JSONPath: `{.peerCount}`,
			},
			{
				Name:     "Expect",
				JSONPath: `{.bootstrapExpect}`,
			},
			{
				Name:     "Leader",
				JSONPath: `{.leader}`,
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

	err := protobuf.RegisterDynamic[OpenGyozaBootstrapStatusSpec](OpenGyozaBootstrapStatusType, &OpenGyozaBootstrapStatus{})
	if err != nil {
		panic(err)
	}
}
