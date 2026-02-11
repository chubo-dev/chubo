// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chuboos

package v1alpha2

import (
	"time"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/siderolabs/go-procfs/procfs"
	"go.uber.org/zap"

	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/block"
	chuboctrl "github.com/siderolabs/talos/internal/app/machined/pkg/controllers/chubo"
	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/config"
	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/cri"
	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/files"
	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/hardware"
	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/network"
	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/perf"
	runtimecontrollers "github.com/siderolabs/talos/internal/app/machined/pkg/controllers/runtime"
	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/secrets"
	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/siderolink"
	timecontrollers "github.com/siderolabs/talos/internal/app/machined/pkg/controllers/time"
	"github.com/siderolabs/talos/internal/app/machined/pkg/controllers/v1alpha1"
	"github.com/siderolabs/talos/internal/app/machined/pkg/runtime"
	"github.com/siderolabs/talos/internal/app/machined/pkg/system"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"github.com/siderolabs/talos/pkg/xfs"
)

func (ctrl *Controller) controllers(
	drainer *runtime.Drainer,
	etcRoot xfs.Root,
	networkEtcRoot xfs.Root,
	networkBindMountTarget string,
	dnsCacheLogger *zap.Logger,
) []controller.Controller {
	return []controller.Controller{
		&block.DevicesController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&block.DiscoveryController{},
		&block.DisksController{},
		&block.LVMActivationController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&block.MountController{},
		&block.MountRequestController{},
		&block.MountStatusController{},
		&block.SwapStatusController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&block.SymlinksController{},
		&block.SystemDiskController{},
		&block.UserDiskConfigController{},
		&block.VolumeConfigController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
			MetaProvider: ctrl.v1alpha1Runtime.State().Machine(),
		},
		&block.VolumeManagerController{},
		&block.ZswapConfigController{},
		&block.ZswapStatusController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&config.AcquireController{
			PlatformConfiguration: &platformConfigurator{
				platform: ctrl.v1alpha1Runtime.State().Platform(),
				state:    ctrl.v1alpha1Runtime.State().V1Alpha2().Resources(),
			},
			PlatformEvent: &platformEventer{
				platform: ctrl.v1alpha1Runtime.State().Platform(),
			},
			Mode:           ctrl.v1alpha1Runtime.State().Platform().Mode(),
			CmdlineGetter:  procfs.ProcCmdline,
			ConfigSetter:   ctrl.v1alpha1Runtime,
			EventPublisher: ctrl.v1alpha1Runtime.Events(),
			ValidationMode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
			ResourceState:  ctrl.v1alpha1Runtime.State().V1Alpha2().Resources(),
		},
		&config.MachineTypeController{},
		&config.PersistenceController{},
		&chuboctrl.BootstrapStatusController{},
		&chuboctrl.OpenWontonServiceController{
			V1Alpha1ServiceManager: system.Services(ctrl.v1alpha1Runtime),
		},
		&chuboctrl.OpenGyozaServiceController{
			V1Alpha1ServiceManager: system.Services(ctrl.v1alpha1Runtime),
		},
		// Installation uses containerd but relies on CRI resource controllers (image cache + registries)
		// to unblock the installer sequence (see v1alpha1 Install task).
		&cri.ImageCacheConfigController{
			V1Alpha1ServiceManager: system.Services(ctrl.v1alpha1Runtime),
		},
		&cri.RegistriesConfigController{},
		&files.EtcFileController{
			EtcRoot: etcRoot,
			EtcPath: "/etc",
		},
		&files.IQNController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&files.NQNController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&hardware.PCIDevicesController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&hardware.PCIDriverRebindConfigController{},
		&hardware.PCIDriverRebindController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&hardware.PCRStatusController{},
		&hardware.SystemInfoController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&network.AddressConfigController{
			Cmdline:      procfs.ProcCmdline(),
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&network.AddressEventController{
			V1Alpha1Events: ctrl.v1alpha1Runtime.Events(),
		},
		network.NewAddressMergeController(),
		&network.AddressSpecController{},
		&network.AddressStatusController{},
		&network.DeviceConfigController{},
		&network.DNSResolveCacheController{
			State:  ctrl.v1alpha1Runtime.State().V1Alpha2().Resources(),
			Logger: dnsCacheLogger,
		},
		&network.DNSUpstreamController{},
		&network.EtcFileController{
			EtcRoot:         networkEtcRoot,
			BindMountTarget: networkBindMountTarget,
			V1Alpha1Mode:    ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&network.EthernetConfigController{},
		&network.EthernetSpecController{},
		&network.EthernetStatusController{},
		&network.HardwareAddrController{},
		&network.HostDNSConfigController{},
		&network.HostnameConfigController{
			Cmdline: procfs.ProcCmdline(),
		},
		network.NewHostnameMergeController(),
		&network.HostnameSpecController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&network.LinkAliasConfigController{},
		&network.LinkAliasSpecController{},
		&network.LinkConfigController{
			Cmdline: procfs.ProcCmdline(),
		},
		network.NewLinkMergeController(),
		&network.LinkSpecController{},
		&network.LinkStatusController{},
		&network.NfTablesChainConfigController{},
		&network.NfTablesChainController{},
		&network.NodeAddressController{},
		&network.NodeAddressSortAlgorithmController{},
		&network.OperatorConfigController{
			Cmdline: procfs.ProcCmdline(),
		},
		network.NewOperatorMergeController(),
		&network.OperatorSpecController{
			V1alpha1Platform: ctrl.v1alpha1Runtime.State().Platform(),
			State:            ctrl.v1alpha1Runtime.State().V1Alpha2().Resources(),
		},
		&network.OperatorVIPConfigController{
			Cmdline: procfs.ProcCmdline(),
		},
		&network.PlatformConfigApplyController{
			V1alpha1Platform: ctrl.v1alpha1Runtime.State().Platform(),
		},
		&network.PlatformConfigController{
			V1alpha1Platform: ctrl.v1alpha1Runtime.State().Platform(),
			PlatformState:    ctrl.v1alpha1Runtime.State().V1Alpha2().Resources(),
		},
		&network.PlatformConfigLoadController{},
		&network.PlatformConfigStoreController{},
		&network.ProbeController{},
		&network.ProbeConfigController{},
		network.NewProbeMergeController(),
		&network.ResolverConfigController{
			Cmdline: procfs.ProcCmdline(),
		},
		network.NewResolverMergeController(),
		&network.ResolverSpecController{},
		&network.RouteConfigController{
			Cmdline: procfs.ProcCmdline(),
		},
		network.NewRouteMergeController(),
		&network.RouteSpecController{},
		&network.RouteStatusController{},
		&network.StatusController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&network.TimeServerConfigController{
			Cmdline: procfs.ProcCmdline(),
		},
		network.NewTimeServerMergeController(),
		&network.TimeServerSpecController{},
		&perf.StatsController{},
		&runtimecontrollers.BootedEntryController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.DevicesStatusController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.DiagnosticsLoggerController{},
		&runtimecontrollers.DropUpgradeFallbackController{
			MetaProvider: ctrl.v1alpha1Runtime.State().Machine(),
		},
		&runtimecontrollers.EnvironmentController{},
		&runtimecontrollers.ExtensionServiceConfigController{},
		&runtimecontrollers.ExtensionServiceConfigFilesController{
			V1Alpha1Mode:            ctrl.v1alpha1Runtime.State().Platform().Mode(),
			ExtensionsConfigBaseDir: constants.ExtensionServiceUserConfigPath,
		},
		&runtimecontrollers.EventsSinkConfigController{
			Cmdline:      procfs.ProcCmdline(),
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.EventsSinkController{
			V1Alpha1Events: ctrl.v1alpha1Runtime.Events(),
			Drainer:        drainer,
		},
		&runtimecontrollers.ExtensionServiceController{
			V1Alpha1Services: system.Services(ctrl.v1alpha1Runtime),
			ConfigPath:       constants.ExtensionServiceConfigPath,
		},
		&runtimecontrollers.ExtensionStatusController{},
		&runtimecontrollers.KernelCmdlineController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.KernelModuleConfigController{},
		&runtimecontrollers.KernelModuleSpecController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.KernelParamConfigController{},
		&runtimecontrollers.KernelParamDefaultsController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.KernelParamSpecController{},
		&runtimecontrollers.KmsgLogConfigController{
			Cmdline: procfs.ProcCmdline(),
		},
		&runtimecontrollers.KmsgLogDeliveryController{
			Drainer: drainer,
		},
		&runtimecontrollers.KmsgLogStorageController{
			V1Alpha1Logging: ctrl.v1alpha1Runtime.Logging(),
			V1Alpha1Mode:    ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.LogPersistenceController{
			V1Alpha1Logging: ctrl.v1alpha1Runtime.Logging(),
		},
		&runtimecontrollers.LoadedKernelModuleController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.MaintenanceConfigController{},
		&runtimecontrollers.MaintenanceServiceController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.MachineStatusPublisherController{
			V1Alpha1Events: ctrl.v1alpha1Runtime.Events(),
		},
		&runtimecontrollers.MountStatusController{},
		&runtimecontrollers.SBOMItemController{},
		&runtimecontrollers.SecurityStateController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&runtimecontrollers.UniqueMachineTokenController{},
		&runtimecontrollers.VersionController{},
		&runtimecontrollers.WatchdogTimerConfigController{},
		&runtimecontrollers.WatchdogTimerController{},
		&runtimecontrollers.OOMController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&secrets.APICertSANsController{},
		&secrets.APIController{},
		&secrets.EncryptionSaltController{},
		&secrets.MaintenanceController{},
		&secrets.MaintenanceCertSANsController{},
		&secrets.MaintenanceRootController{},
		secrets.NewRootOSController(),
		&secrets.TrustedRootsController{},
		&secrets.TrustdController{},
		&siderolink.ConfigController{
			Cmdline:      procfs.ProcCmdline(),
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&siderolink.ManagerController{},
		&siderolink.StatusController{},
		&siderolink.UserspaceWireguardController{
			RelayRetryTimeout: 10 * time.Second,
		},
		&timecontrollers.AdjtimeStatusController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&timecontrollers.SyncController{
			V1Alpha1Mode: ctrl.v1alpha1Runtime.State().Platform().Mode(),
		},
		&v1alpha1.ServiceController{
			V1Alpha1Events: ctrl.v1alpha1Runtime.Events(),
		},
	}
}
