// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generate

import (
	"fmt"

	"github.com/siderolabs/go-pointer"

	"github.com/chubo-dev/chubo/pkg/machinery/config/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	v1alpha1 "github.com/chubo-dev/chubo/pkg/machinery/config/types/v1alpha1"
)

//nolint:gocyclo,cyclop
func (in *Input) init() ([]config.Document, error) {
	v1alpha1Config := &v1alpha1.Config{
		ConfigVersion: "v1alpha1",
		ConfigDebug:   pointer.To(in.Options.Debug),
		ConfigPersist: pointer.To(true),
	}

	machineConfig := &v1alpha1.MachineConfig{
		MachineType:     machine.TypeInit.String(),
		MachineCA:       in.Options.SecretsBundle.Certs.OS,
		MachineCertSANs: in.AdditionalMachineCertSANs,
		MachineToken:    in.Options.SecretsBundle.TrustdInfo.Token,
		MachineInstall: &v1alpha1.InstallConfig{
			InstallDisk:            in.Options.InstallDisk,
			InstallImage:           in.Options.InstallImage,
			InstallWipe:            pointer.To(false),
			InstallExtraKernelArgs: in.Options.InstallExtraKernelArgs,
		},
		MachineDisks:    in.Options.MachineDisks,
		MachineSysctls:  in.Options.Sysctls,
		MachineFeatures: &v1alpha1.FeaturesConfig{},
	}

	if in.Options.VersionContract.GrubUseUKICmdlineDefault() {
		machineConfig.MachineInstall.InstallGrubUseUKICmdline = pointer.To(true)
	}

	if !in.Options.VersionContract.HideRBACAndKeyUsage() {
		machineConfig.MachineFeatures.RBAC = pointer.To(true)

		if in.Options.VersionContract.ApidExtKeyUsageCheckEnabled() {
			machineConfig.MachineFeatures.ApidCheckExtKeyUsage = pointer.To(true)
		}
	}

	if in.Options.VersionContract.DiskQuotaSupportEnabled() {
		machineConfig.MachineFeatures.DiskQuotaSupport = pointer.To(true)
	}

	cluster := &v1alpha1.ClusterConfig{
		ClusterID:     in.Options.SecretsBundle.Cluster.ID,
		ClusterName:   in.ClusterName,
		ClusterSecret: in.Options.SecretsBundle.Cluster.Secret,
	}

	if in.Options.DiscoveryEnabled != nil {
		cluster.ClusterDiscoveryConfig = &v1alpha1.ClusterDiscoveryConfig{
			DiscoveryEnabled: pointer.To(*in.Options.DiscoveryEnabled),
		}
	}

	v1alpha1Config.MachineConfig = machineConfig
	v1alpha1Config.ClusterConfig = cluster

	documents := []config.Document{v1alpha1Config}

	extraDocuments, err := in.generateRegistryConfigs(machineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate registry configs: %w", err)
	}

	documents = append(documents, extraDocuments...)

	extraDocuments, err = in.generateNetworkConfigs(machineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate network configs: %w", err)
	}

	documents = append(documents, extraDocuments...)

	return documents, nil
}

func ptrOrNil(b bool) *bool {
	if b {
		return &b
	}

	return nil
}
