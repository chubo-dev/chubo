// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package chuboos is a compatibility shim.
//
// New code should import github.com/chubo-dev/chubo/pkg/machinery/config/types/chubo.
package chuboos

import chubo "github.com/chubo-dev/chubo/pkg/machinery/config/types/chubo"

const (
	MachineConfigKind               = chubo.MachineConfigKind
	MachineConfigAPIVersion         = chubo.MachineConfigAPIVersion
	ChuboBootstrapModeSignedPayload = chubo.ChuboBootstrapModeSignedPayload
)

type (
	MachineConfigV1Alpha1 = chubo.MachineConfigV1Alpha1
	MachineConfigMetadata = chubo.MachineConfigMetadata
	MachineConfigSpec     = chubo.MachineConfigSpec
	InstallSpec           = chubo.InstallSpec
	NetworkSpec           = chubo.NetworkSpec
	TimeSpec              = chubo.TimeSpec
	LoggingSpec           = chubo.LoggingSpec
	TrustSpec             = chubo.TrustSpec
	CASpec                = chubo.CASpec
	RegistrySpec          = chubo.RegistrySpec
	RegistryMirrorSpec    = chubo.RegistryMirrorSpec
	ModulesSpec           = chubo.ModulesSpec
	ChuboModuleSpec       = chubo.ChuboModuleSpec
	ChuboBootstrapSpec    = chubo.ChuboBootstrapSpec
	ChuboRoleSpec         = chubo.ChuboRoleSpec
	ChuboOpenBaoSpec      = chubo.ChuboOpenBaoSpec
	BootstrapSpec         = chubo.BootstrapSpec
)

func NewMachineConfigV1Alpha1() *MachineConfigV1Alpha1 {
	return chubo.NewMachineConfigV1Alpha1()
}
