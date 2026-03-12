// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package secrets

import (
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/meta"
	"github.com/cosi-project/runtime/pkg/resource/protobuf"
	"github.com/cosi-project/runtime/pkg/resource/typed"

	"github.com/chubo-dev/chubo/pkg/machinery/proto"
)

// OpenBaoInitType is the type of OpenBao bootstrap material resource.
const OpenBaoInitType = resource.Type("OpenBaoInits.secrets.talos.dev")

// OpenBaoInitID is the singleton ID for host-native OpenBao bootstrap material.
const OpenBaoInitID = resource.ID("openbao")

// OpenBaoInit stores the host-native OpenBao init response.
type OpenBaoInit = typed.Resource[OpenBaoInitSpec, OpenBaoInitExtension]

// OpenBaoInitSpec contains sensitive OpenBao bootstrap material.
//
//gotagsrewrite:gen
type OpenBaoInitSpec struct {
	RootToken  string   `yaml:"rootToken,omitempty" protobuf:"1"`
	KeysBase64 []string `yaml:"keysBase64,omitempty" protobuf:"2"`
}

// DeepCopy returns a deep copy of OpenBaoInitSpec.
func (o OpenBaoInitSpec) DeepCopy() OpenBaoInitSpec {
	out := OpenBaoInitSpec{
		RootToken: o.RootToken,
	}

	if len(o.KeysBase64) > 0 {
		out.KeysBase64 = append([]string(nil), o.KeysBase64...)
	}

	return out
}

// NewOpenBaoInit initializes an OpenBaoInit resource.
func NewOpenBaoInit() *OpenBaoInit {
	return typed.NewResource[OpenBaoInitSpec, OpenBaoInitExtension](
		resource.NewMetadata(NamespaceName, OpenBaoInitType, OpenBaoInitID, resource.VersionUndefined),
		OpenBaoInitSpec{},
	)
}

// OpenBaoInitExtension provides auxiliary methods for OpenBaoInit.
type OpenBaoInitExtension struct{}

// ResourceDefinition implements [typed.Extension].
func (OpenBaoInitExtension) ResourceDefinition() meta.ResourceDefinitionSpec {
	return meta.ResourceDefinitionSpec{
		Type:             OpenBaoInitType,
		Aliases:          []resource.Type{"openbaoinit"},
		DefaultNamespace: NamespaceName,
		Sensitivity:      meta.Sensitive,
	}
}

func init() {
	proto.RegisterDefaultTypes()

	if err := protobuf.RegisterDynamic[OpenBaoInitSpec](OpenBaoInitType, &OpenBaoInit{}); err != nil {
		panic(err)
	}
}
