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

// OpenBaoJobStatusType is the type of OpenBaoJobStatus resource.
const OpenBaoJobStatusType = resource.Type("OpenBaoJobStatuses.chubo.dev")

// OpenBaoJobStatusID is the single ID for the OpenBaoJobStatus resource.
const OpenBaoJobStatusID = resource.ID("openbao")

// OpenBaoJobStatus reports the state of OpenBao Nomad job management.
type OpenBaoJobStatus = typed.Resource[OpenBaoJobStatusSpec, OpenBaoJobStatusExtension]

// OpenBaoJobStatusSpec describes OpenBao job status.
//
//gotagsrewrite:gen
type OpenBaoJobStatusSpec struct {
	// Configured indicates whether machine config requested OpenBao as a Nomad job.
	Configured bool `yaml:"configured" protobuf:"1"`
	// Mode is the requested openbao mode.
	Mode string `yaml:"mode,omitempty" protobuf:"2"`
	// JobID is the target Nomad job ID.
	JobID string `yaml:"jobID,omitempty" protobuf:"3"`
	// NomadReachable indicates whether Nomad API was reachable.
	NomadReachable bool `yaml:"nomadReachable" protobuf:"4"`
	// Present indicates whether the OpenBao job is present in Nomad.
	Present bool `yaml:"present" protobuf:"5"`
	// LastError is a last reconciliation error, if any.
	LastError string `yaml:"lastError,omitempty" protobuf:"6"`
}

// DeepCopy generates a deep copy of OpenBaoJobStatusSpec.
func (o OpenBaoJobStatusSpec) DeepCopy() OpenBaoJobStatusSpec {
	return o
}

// NewOpenBaoJobStatus initializes an OpenBaoJobStatus resource.
func NewOpenBaoJobStatus() *OpenBaoJobStatus {
	return typed.NewResource[OpenBaoJobStatusSpec, OpenBaoJobStatusExtension](
		resource.NewMetadata(NamespaceName, OpenBaoJobStatusType, OpenBaoJobStatusID, resource.VersionUndefined),
		OpenBaoJobStatusSpec{},
	)
}

// OpenBaoJobStatusExtension provides auxiliary methods for OpenBaoJobStatus.
type OpenBaoJobStatusExtension struct{}

// ResourceDefinition implements [typed.Extension] interface.
func (OpenBaoJobStatusExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type: OpenBaoJobStatusType,
		Aliases: []resource.Type{
			"openbaojobstatus",
			"openbaojob",
		},
		DefaultNamespace: NamespaceName,
		PrintColumns: []meta.PrintColumn{
			{
				Name:     "Configured",
				JSONPath: `{.configured}`,
			},
			{
				Name:     "Mode",
				JSONPath: `{.mode}`,
			},
			{
				Name:     "JobID",
				JSONPath: `{.jobID}`,
			},
			{
				Name:     "NomadReachable",
				JSONPath: `{.nomadReachable}`,
			},
			{
				Name:     "Present",
				JSONPath: `{.present}`,
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

	err := protobuf.RegisterDynamic[OpenBaoJobStatusSpec](OpenBaoJobStatusType, &OpenBaoJobStatus{})
	if err != nil {
		panic(err)
	}
}
