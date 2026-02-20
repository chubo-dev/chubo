// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package constants defines constants used throughout Talos.
package constants

import (
	"time"

	"github.com/siderolabs/crypto/x509"
)

const (
	// DefaultKernelVersion is the default Linux kernel version.
	DefaultKernelVersion = "6.18.8-talos"

	// KernelParamConfig is the kernel parameter name for specifying the URL.
	// to the config.
	KernelParamConfig = "talos.config"

	// KernelParamConfigInline is the kernel parameter name for specifying the inline config.
	//
	// The inline config should be base64 encoded and zstd-compressed.
	KernelParamConfigInline = "talos.config.inline"

	// KernelParamConfigEarly is the kernel parameter name for specifying the inline config (as the first source).
	//
	// The inline config should be base64 encoded and zstd-compressed.
	KernelParamConfigEarly = "talos.config.early"

	// KernelParamConfigOAuthClientID is the kernel parameter name for specifying the OAuth2 client ID.
	KernelParamConfigOAuthClientID = "talos.config.oauth.client_id"

	// KernelParamConfigOAuthClientSecret is the kernel parameter name for specifying the OAuth2 client secret.
	KernelParamConfigOAuthClientSecret = "talos.config.oauth.client_secret"

	// KernelParamConfigOAuthAudience is the kernel parameter name for specifying the OAuth2 audience.
	KernelParamConfigOAuthAudience = "talos.config.oauth.audience"

	// KernelParamConfigOAuthScope is the kernel parameter name for specifying the OAuth2 scopes (might be repeated).
	KernelParamConfigOAuthScope = "talos.config.oauth.scope"

	// KernelParamConfigOAuthDeviceAuthURL is the kernel parameter name for specifying the OAuth2 device auth URL.
	KernelParamConfigOAuthDeviceAuthURL = "talos.config.oauth.device_auth_url"

	// KernelParamConfigOAuthTokenURL is the kernel parameter name for specifying the OAuth2 token URL.
	KernelParamConfigOAuthTokenURL = "talos.config.oauth.token_url"

	// KernelParamConfigOAuthExtraVariable is the kernel parameter name for specifying the OAuth2 extra variable (might be repeated).
	KernelParamConfigOAuthExtraVariable = "talos.config.oauth.extra_variable"

	// ConfigNone indicates no config is required.
	ConfigNone = "none"

	// KernelParamPlatform is the kernel parameter name for specifying the
	// platform.
	KernelParamPlatform = "talos.platform"

	// KernelParamEventsSink is the kernel parameter name for specifying the
	// events sink server.
	KernelParamEventsSink = "talos.events.sink"

	// KernelParamLoggingKernel is the kernel parameter name for specifying the
	// kernel log delivery destination.
	KernelParamLoggingKernel = "talos.logging.kernel"

	// KernelParamWipe is the kernel parameter name for specifying the
	// disk to wipe on the next boot and reboot.
	KernelParamWipe = "talos.experimental.wipe"

	// KernelParamDeviceSettleTime is the kernel parameter name for specifying the
	// extra device settle timeout.
	KernelParamDeviceSettleTime = "talos.device.settle_time"

	// KernelParamCGroups is the legacy kernel parameter not supported anymore.
	KernelParamCGroups = "talos.unified_cgroup_hierarchy"

	// KernelParamAuditdDisabled is the kernel parameter name for disabling auditd service.
	KernelParamAuditdDisabled = "talos.auditd.disabled"

	// KernelParamDashboardDisabled is the kernel parameter name for disabling the dashboard.
	KernelParamDashboardDisabled = "talos.dashboard.disabled"

	// KernelParamDashboardConsole is the kernel parameter name for specifying the dashboard console.
	KernelParamDashboardConsole = "talos.dashboard.console"

	// KernelParamEnvironment is the kernel parameter name for passing process environment.
	KernelParamEnvironment = "talos.environment"

	// KernelParamNetIfnames is the kernel parameter name to control predictable network interface names.
	KernelParamNetIfnames = "net.ifnames"

	// KernelParamHaltIfInstalled is the kernel parameter name to control if Talos should pause if booting from boot media while Talos is already installed.
	KernelParamHaltIfInstalled = "talos.halt_if_installed"

	// KernelParamSELinux is the kernel parameter name to enable/disable SELinux.
	KernelParamSELinux = "selinux"

	// KernelParamSELinuxEnforcing is the kernel parameter name to control SELinux enforcement mode.
	KernelParamSELinuxEnforcing = "enforcing"

	// KernelParamHostname is the kernel parameter name for specifying the
	// hostname.
	KernelParamHostname = "talos.hostname"

	// KernelParamShutdown is the kernel parameter for specifying the
	// shutdown type (halt/poweroff).
	KernelParamShutdown = "talos.shutdown"

	// KernelParamNetworkInterfaceIgnore is the kernel parameter for specifying network interfaces which should be ignored by talos.
	KernelParamNetworkInterfaceIgnore = "talos.network.interface.ignore"

	// KernelParamVlan is the kernel parameter for specifying vlan for the interface.
	KernelParamVlan = "vlan"

	// KernelParamBonding is the kernel parameter for specifying bonded network interfaces.
	KernelParamBonding = "bond"

	// KernelParamPanic is the kernel parameter name for specifying the time to wait until rebooting after kernel panic (0 disables reboot).
	KernelParamPanic = "panic"

	// KernelParamSideroLink is the kernel parameter name to specify SideroLink API endpoint.
	KernelParamSideroLink = "siderolink.api"

	// KernelParamEquinixMetalEvents is the kernel parameter name to specify the Equinix Metal phone home endpoint.
	// This param is injected by Equinix Metal and depends on the device ID and datacenter.
	KernelParamEquinixMetalEvents = "em.events_url"

	// KernelParamEnforceModuleSigVerify is the kernel parameter name to specify module signature verification enforcement.
	// see https://github.com/chubo-dev/chubo/issues/11989
	KernelParamEnforceModuleSigVerify = "module.sig_enforce"

	// NewRoot is the path where the switchroot target is mounted.
	NewRoot = "/root"

	// ExtensionLayers is the path where the extensions layers are stored.
	ExtensionLayers = "/layers"

	// ExtensionsConfigFile is the extensions layers configuration file name in the initramfs.
	ExtensionsConfigFile = "/extensions.yaml"

	// ExtensionsRuntimeConfigFile extensions layers configuration file name in the rootfs.
	ExtensionsRuntimeConfigFile = "/etc/extensions.yaml"

	// EFIPartitionLabel is the label of the partition to use for mounting at
	// the boot path.
	EFIPartitionLabel = "EFI"

	// EFIMountPoint is the label of the partition to use for mounting at
	// the boot path.
	EFIMountPoint = BootMountPoint + "/EFI"

	// EFIVarsMountPoint is mount point for efivars filesystem type.
	// https://www.kernel.org/doc/html/next/filesystems/efivarfs.html
	EFIVarsMountPoint = "/sys/firmware/efi/efivars"

	// BIOSGrubPartitionLabel is the label of the partition used by grub's second
	// stage bootloader.
	BIOSGrubPartitionLabel = "BIOS"

	// MetaPartitionLabel is the label of the meta partition.
	MetaPartitionLabel = "META"

	// StatePartitionLabel is the label of the state partition.
	StatePartitionLabel = "STATE"

	// StateMountPoint is the label of the partition to use for mounting at
	// the state path.
	StateMountPoint = "/system/state"

	// StateSelinuxLabel is the label to be assigned to the state mount.
	StateSelinuxLabel = "system_u:object_r:system_state_t:s0"

	// BootPartitionLabel is the label of the partition to use for mounting at
	// the boot path.
	BootPartitionLabel = "BOOT"

	// BootMountPoint is the label of the partition to use for mounting at
	// the boot path.
	BootMountPoint = "/boot"

	// EphemeralPartitionLabel is the label of the partition to use for
	// mounting at the data path.
	EphemeralPartitionLabel = "EPHEMERAL"

	// EphemeralMountPoint is the label of the partition to use for mounting at
	// the data path.
	EphemeralMountPoint = "/var"

	// EphemeralSelinuxLabel is the label to be assigned to the ephemeral mount.
	EphemeralSelinuxLabel = "system_u:object_r:ephemeral_t:s0"

	// OptSELinuxLabel is the SELinux label to be set for /opt overlay mount.
	OptSELinuxLabel = "system_u:object_r:opt_t:s0"

	// RootMountPoint is the label of the partition to use for mounting at
	// the root path.
	RootMountPoint = "/"

	// ISOFilesystemLabel is the label of the ISO file system for the Talos
	// installer.
	ISOFilesystemLabel = "TALOS"

	// WorkloadShutdownGracePeriod is the shutdown grace period for workload containers.
	WorkloadShutdownGracePeriod = 30 * time.Second

	// WorkloadSeccompProfilesDirectory is where user-provided seccomp profiles are mounted.
	WorkloadSeccompProfilesDirectory = "/var/lib/workload/seccomp/profiles"

	// DefaultWorkloadVersion is the legacy default version used by image helper commands.
	DefaultWorkloadVersion = "1.35.0"

	// DefaultControlPlanePort is the default port to use for the control plane.
	DefaultControlPlanePort = 6443

	// CoreDNSImage is the enforced CoreDNS image to use.
	CoreDNSImage = "docker.io/coredns/coredns"

	// DefaultCoreDNSVersion is the default version for the CoreDNS.
	// renovate: datasource=docker depName=docker.io/coredns/coredns
	DefaultCoreDNSVersion = "1.13.2"

	// ConfigFilename is the filename of the saved config in STATE partition.
	ConfigFilename = "config.yaml"

	// EmbeddedConfigDirectory is the path to the embedded config is placed inside rootfs.
	EmbeddedConfigDirectory = "/usr/local/etc/talos/"

	// ConfigTryTimeout is the timeout of the config apply in try mode.
	ConfigTryTimeout = time.Minute

	// MetalConfigISOLabel is the volume label for ISO based configuration.
	MetalConfigISOLabel = "metal-iso"

	// ConfigGuestInfo is the name of the VMware guestinfo config strategy.
	ConfigGuestInfo = "guestinfo"

	// VMwareGuestInfoPrefix is the prefix to extraConfig variables.
	VMwareGuestInfoPrefix = "guestinfo."

	// VMwareGuestInfoConfigKey is the guestinfo key used to provide a config file.
	VMwareGuestInfoConfigKey = "talos.config"

	// VMwareGuestInfoFallbackKey is the fallback guestinfo key used to provide a config file.
	VMwareGuestInfoFallbackKey = "userdata"

	// VMwareGuestInfoMetadataKey is the guestinfo key used to provide metadata.
	VMwareGuestInfoMetadataKey = "metadata"

	// VMwareGuestInfoOvfEnvKey is the guestinfo key used to provide the OVF environment.
	VMwareGuestInfoOvfEnvKey = "ovfenv"

	// ApidPort is the port for the apid service.
	ApidPort = 50000

	// ApidUserID is the user ID for apid.
	ApidUserID = 50

	// DashboardUserID is the user ID for dashboard.
	// We use the same user ID as apid so that the dashboard can write to the machined unix socket.
	DashboardUserID = ApidUserID

	// DashboardPriority is the priority for the dashboard service.
	// Higher nice value for the dashboard to give more CPU time to other services when under load.
	DashboardPriority = 10

	// TrustdPort is the port for the trustd service.
	TrustdPort = 50001

	// TrustdUserID is the user ID for trustd.
	TrustdUserID = 51

	// DefaultContainerdVersion is the default container runtime version.
	DefaultContainerdVersion = "2.2.1"

	// RuncVersion is the runc version.
	RuncVersion = "1.4.0"

	// SystemContainerdNamespace is the Containerd namespace for Talos services.
	SystemContainerdNamespace = "system"

	// SystemContainerdAddress is the path to the system containerd socket.
	SystemContainerdAddress = SystemRunPath + "/containerd/containerd.sock"

	// WorkloadContainerdNamespace is the containerd namespace for workload tasks.
	WorkloadContainerdNamespace = "workload"

	// WorkloadContainerdAddress is the path to the workload containerd socket.
	WorkloadContainerdAddress = "/run/containerd/containerd.sock"

	// EtcRuntimeConfdPath is the path to the directory providing runtime plugin config fragments.
	EtcRuntimeConfdPath = "/etc/workload/conf.d"

	// RuntimeConfdPath is the path to the runtime config fragments relative to /etc.
	RuntimeConfdPath = "workload/conf.d"

	// RuntimeConfig is the path to the merged runtime configuration file relative to /etc.
	RuntimeConfig = "workload/conf.d/runtime.toml"

	// RuntimeRegistryConfigPart is the path to the generated registry config fragment relative to /etc.
	RuntimeRegistryConfigPart = "workload/conf.d/01-registries.part"

	// RuntimeCustomizationConfigPart is the path to the generated customization config fragment relative to /etc.
	RuntimeCustomizationConfigPart = "workload/conf.d/20-customization.part"

	// RuntimeBaseSpec is the path to the base runtime specification fragment.
	RuntimeBaseSpec = "workload/conf.d/base-spec.json"

	// ChuboConfigEnvVar is the environment variable for setting the Chubo configuration file path.
	ChuboConfigEnvVar = "CHUBOCONFIG"

	// TalosConfigEnvVar is the legacy environment variable for setting the Talos configuration file path.
	TalosConfigEnvVar = "TALOSCONFIG"

	// ChuboHomeEnvVar is the environment variable for setting the Chubo state directory file path.
	ChuboHomeEnvVar = "CHUBO_HOME"

	// TalosHomeEnvVar is the legacy environment variable for setting the Talos state directory file path.
	TalosHomeEnvVar = "TALOS_HOME"

	// APISocketPath is the path to file socket of apid.
	APISocketPath = SystemRunPath + "/apid/apid.sock"

	// APISocketLabel is the SELinux label for apid socket file.
	APISocketLabel = "system_u:object_r:apid_socket_t:s0"

	// APIRuntimeSocketPath is the path to file socket of runtime server for apid.
	APIRuntimeSocketPath = SystemRunPath + "/apid/runtime.sock"

	// APIRuntimeSocketLabel is the SELinux label for apid runtime socket file.
	APIRuntimeSocketLabel = "system_u:object_r:apid_runtime_socket_t:s0"

	// TrustdRuntimeSocketPath is the path to file socket of runtime server for trustd.
	TrustdRuntimeSocketPath = SystemRunPath + "/trustd/runtime.sock"

	// TrustdRuntimeSocketLabel is the SELinux label for trustd runtime socket file.
	TrustdRuntimeSocketLabel = "system_u:object_r:trustd_runtime_socket_t:s0"

	// MachineSocketPath is the path to file socket of machine API.
	MachineSocketPath = SystemRunPath + "/machined/machine.sock"

	// MachineSocketLabel is the SELinux label for socket of machine API.
	MachineSocketLabel = "system_u:object_r:machine_socket_t:s0"

	// NetworkSocketPath is the path to file socket of network API.
	NetworkSocketPath = SystemRunPath + "/networkd/networkd.sock"

	// ArchVariable is replaced automatically by the target cluster arch.
	ArchVariable = "${ARCH}"

	// KernelAsset defines a well known name for our kernel filename.
	KernelAsset = "vmlinuz"

	// KernelAssetWithArch defines a well known name for our kernel filename with arch variable.
	KernelAssetWithArch = "vmlinuz-" + ArchVariable

	// KernelAssetPath is the path to the kernel on disk.
	KernelAssetPath = "/usr/install/%s/" + KernelAsset

	// InitramfsAsset defines a well known name for our initramfs filename.
	InitramfsAsset = "initramfs.xz"

	// InitramfsAssetWithArch defines a well known name for our initramfs filename with arch variable.
	InitramfsAssetWithArch = "initramfs-" + ArchVariable + ".xz"

	// InitramfsAssetPath is the path to the initramfs on disk.
	InitramfsAssetPath = "/usr/install/%s/" + InitramfsAsset

	// RootfsAsset defines a well known name for our rootfs filename.
	RootfsAsset = "rootfs.sqsh"

	// UKIAsset defines a well known name for our UKI filename.
	UKIAsset = "vmlinuz.efi"

	// UKIAssetPath is the path to the UKI in the installer.
	UKIAssetPath = "/usr/install/%s/" + UKIAsset

	// SDStubAsset defines a well known name for our systemd-stub filename.
	SDStubAsset = "systemd-stub.efi"

	// SDStubAssetPath is the path to the systemd-stub in the installer.
	SDStubAssetPath = "/usr/install/%s/" + SDStubAsset

	// SDBootAsset defines a well known name for our SDBoot filename.
	SDBootAsset = "systemd-boot.efi"

	// SDBootAssetPath is the path to the SDBoot in the installer.
	SDBootAssetPath = "/usr/install/%s/" + SDBootAsset

	// ImagerOverlayBasePath is the base path for the imager overlay.
	ImagerOverlayBasePath = "/overlay"
	// ImagerOverlayArtifactsPath is the path to the artifacts in the imager overlay.
	ImagerOverlayArtifactsPath = ImagerOverlayBasePath + "/" + "artifacts"
	// ImagerOverlayInstallersPath is the path to the installers in the imager overlay.
	ImagerOverlayInstallersPath = ImagerOverlayBasePath + "/" + "installers"
	// ImagerOverlayProfilesPath is the path to the profiles in the imager overlay.
	ImagerOverlayProfilesPath = ImagerOverlayBasePath + "/" + "profiles"
	// ImagerOverlayInstallerDefault is the default installer name.
	ImagerOverlayInstallerDefault = "default"
	// ImagerOverlayInstallerDefaultPath is the path to the default installer in the imager overlay.
	ImagerOverlayInstallerDefaultPath = ImagerOverlayInstallersPath + "/" + ImagerOverlayInstallerDefault
	// ImagerOverlayExtraOptionsPath is the path to the generated extra options file in the imager overlay.
	ImagerOverlayExtraOptionsPath = ImagerOverlayBasePath + "/" + "extra-options"

	// PlatformKeyAsset defines a well known name for the platform key filename used for auto-enrolling.
	PlatformKeyAsset = "PK.auth"

	// KeyExchangeKeyAsset defines a well known name for the key exchange key filename used for auto-enrolling.
	KeyExchangeKeyAsset = "KEK.auth"

	// SignatureKeyAsset defines a well known name for the signature key filename used for auto-enrolling.
	SignatureKeyAsset = "db.auth"

	// SecureBootSigningKeyAsset defines a well known name for the secure boot signing key filename.
	SecureBootSigningKeyAsset = "uki-signing-key.pem"

	// SecureBootSigningCertAsset defines a well known name for the secure boot signing key filename.
	SecureBootSigningCertAsset = "uki-signing-cert.pem"

	// PCRSigningKeyAsset defines a well known name for the PCR signing key filename.
	PCRSigningKeyAsset = "pcr-signing-key.pem"

	// SDStubDynamicInitrdPath is the path where dynamically generated initrds are placed by systemd-stub.
	// https://www.mankier.com/7/systemd-stub#Description
	SDStubDynamicInitrdPath = "/.extra"

	// PCRSignatureJSON is the path to the PCR signature JSON file.
	// https://www.mankier.com/7/systemd-stub#Initrd_Resources
	PCRSignatureJSON = SDStubDynamicInitrdPath + "/" + "tpm2-pcr-signature.json"

	// PCRPublicKey is the path to the PCR public key file.
	// https://www.mankier.com/7/systemd-stub#Initrd_Resources
	PCRPublicKey = SDStubDynamicInitrdPath + "/" + "tpm2-pcr-public-key.pem"

	// UKIPCR is the PCR number where systemd-stub measures the UKI.
	UKIPCR = 11

	// SecureBootStatePCR is the PCR number where the secure boot state and the signature are measured.
	// PCR 7 changes when UEFI SecureBoot mode is enabled/disabled, or firmware certificates (PK, KEK, db, dbx, …) are updated.
	SecureBootStatePCR = 7

	// DefaultCertificateValidityDuration is the default duration for a certificate.
	DefaultCertificateValidityDuration = x509.DefaultCertificateValidityDuration

	// SystemPath is the path to write temporary runtime system related files
	// and directories.
	SystemPath = "/system"

	// SystemSelinuxLabel is the SELinux label for runtime system related files and directories.
	SystemSelinuxLabel = "system_u:object_r:system_t:s0"

	// RunPath is the path to the system run directory.
	RunPath = "/run"

	// RunSelinuxLabel is the SELinux label for the run directory.
	RunSelinuxLabel = "system_u:object_r:run_t:s0"

	// VarSystemOverlaysPath is the path where overlay mounts are created.
	VarSystemOverlaysPath = "/var/system/overlays"

	// SystemRunPath is the path to the system run directory.
	SystemRunPath = SystemPath + "/run"

	// SystemVarPath is the path to the system var directory.
	SystemVarPath = SystemPath + "/var"

	// SystemVarSelinuxLabel is the SELinux label for the system var directory.
	SystemVarSelinuxLabel = "system_u:object_r:system_var_t:s0"

	// SystemEtcPath is the path to the system etc directory.
	SystemEtcPath = SystemPath + "/etc"

	// EtcSelinuxLabel is the SELinux label for the /etc and /system/etc directories.
	EtcSelinuxLabel = "system_u:object_r:etc_t:s0"

	// SystemLibexecPath is the path to the system libexec directory.
	SystemLibexecPath = SystemPath + "/libexec"

	// SystemOverlaysPath is the path to the system overlay directory.
	SystemOverlaysPath = SystemPath + "/overlays"

	// CgroupMountPath is the default mount path for unified cgroupsv2 setup.
	CgroupMountPath = "/sys/fs/cgroup"

	// CgroupInit is the cgroup name for init process.
	CgroupInit = "/init"

	// CgroupInitReservedMemory is the hard memory protection for the init process.
	CgroupInitReservedMemory = 96 * 1024 * 1024

	// CgroupInitMillicores is the CPU weight for the init process.
	CgroupInitMillicores = 2000

	// CgroupSystem is the cgroup name for system processes.
	CgroupSystem = "/system"

	// CgroupSystemMillicores is the CPU weight for the system cgroup.
	CgroupSystemMillicores = 1500

	// CgroupSystemReservedMemory is the hard memory protection for the system processes.
	CgroupSystemReservedMemory = 96 * 1024 * 1024

	// CgroupSystemRuntime is the cgroup name for containerd runtime processes.
	CgroupSystemRuntime = CgroupSystem + "/runtime"

	// CgroupSystemRuntimeReservedMemory is the hard memory protection for the system containerd process.
	CgroupSystemRuntimeReservedMemory = 48 * 1024 * 1024

	// CgroupSystemRuntimeMillicores is the CPU weight for the system containerd process.
	CgroupSystemRuntimeMillicores = 500

	// CgroupSystemDebug is the cgroup name for debug processes.
	CgroupSystemDebug = CgroupSystem + "/debug"

	// SelinuxLabelMachined is the SELinux label for machined.
	SelinuxLabelMachined = "system_u:system_r:init_t:s0"

	// SelinuxLabelInstaller is the SELinux label for the installer.
	SelinuxLabelInstaller = "system_u:system_r:installer_t:s0"

	// SelinuxLabelUnconfinedSysContainer is the SELinux label for system containers without label set (normally extensions).
	SelinuxLabelUnconfinedSysContainer = "system_u:system_r:unconfined_container_t:s0"

	// SelinuxLabelUnconfinedService is the SELinux label for process without label set (normally should not occur).
	SelinuxLabelUnconfinedService = "system_u:system_r:unconfined_service_t:s0"

	// SelinuxLabelSystemRuntime is the SELinux label for containerd runtime processes.
	SelinuxLabelSystemRuntime = "system_u:system_r:sys_containerd_t:s0"

	// CgroupApid is the cgroup name for apid runtime processes.
	CgroupApid = CgroupSystem + "/apid"

	// CgroupApidReservedMemory is the hard memory protection for the apid processes.
	CgroupApidReservedMemory = 16 * 1024 * 1024

	// CgroupApidMaxMemory is the hard memory limit for the apid process.
	CgroupApidMaxMemory = 128 * 1024 * 1024

	// CgroupApidMillicores is the CPU weight for the apid process.
	CgroupApidMillicores = 500

	// SelinuxLabelApid is the SELinux label for apid runtime processes.
	SelinuxLabelApid = "system_u:system_r:apid_t:s0"

	// CgroupTrustd is the cgroup name for trustd runtime processes.
	CgroupTrustd = CgroupSystem + "/trustd"

	// CgroupTrustdReservedMemory is the hard memory protection for the trustd processes.
	CgroupTrustdReservedMemory = 8 * 1024 * 1024

	// CgroupTrustdMaxMemory is the hard memory limit for the trustd process.
	CgroupTrustdMaxMemory = 128 * 1024 * 1024

	// CgroupTrustdMillicores is the CPU weight for the trustd process.
	CgroupTrustdMillicores = 250

	// SelinuxLabelTrustd is the SELinux label for trustd runtime processes.
	SelinuxLabelTrustd = "system_u:system_r:trustd_t:s0"

	// CgroupUdevd is the cgroup name for udevd runtime processes.
	CgroupUdevd = CgroupSystem + "/udevd"

	// CgroupUdevdReservedMemory is the hard memory protection for the udevd processes.
	CgroupUdevdReservedMemory = 8 * 1024 * 1024

	// CgroupUdevdMillicores is the CPU weight for the udevd process.
	CgroupUdevdMillicores = 250

	// SelinuxLabelUdevd is the SELinux label for udevd runtime processes.
	SelinuxLabelUdevd = "system_u:system_r:udev_t:s0"

	// CgroupExtensions is the cgroup name for system extension processes.
	CgroupExtensions = CgroupSystem + "/extensions"

	// CgroupDashboard is the cgroup name for dashboard process.
	CgroupDashboard = CgroupSystem + "/dashboard"

	// SelinuxLabelDashboard is the SELinux label for dashboard process.
	SelinuxLabelDashboard = "system_u:system_r:dashboard_t:s0"

	// CgroupPodRuntimeRoot is the cgroup containing workload runtime components.
	CgroupPodRuntimeRoot = "/podruntime"

	// CgroupWorkloadsRoot is the cgroup root for workload QoS classes.
	CgroupWorkloadsRoot = "/workloads"

	// CgroupWorkloadBestEffort is the cgroup for best-effort workload pods.
	CgroupWorkloadBestEffort = CgroupWorkloadsRoot + "/besteffort"

	// CgroupWorkloadBurstable is the cgroup for burstable workload pods.
	CgroupWorkloadBurstable = CgroupWorkloadsRoot + "/burstable"

	// CgroupWorkloadGuaranteed is the cgroup for guaranteed workload pods.
	CgroupWorkloadGuaranteed = CgroupWorkloadsRoot + "/guaranteed"

	// CgroupPodRuntimeRootMillicores is the CPU weight for the pod runtime cgroup.
	CgroupPodRuntimeRootMillicores = 4000

	// CgroupPodRuntime is the cgroup name for workload runtime processes.
	CgroupPodRuntime = CgroupPodRuntimeRoot + "/runtime"

	// CgroupPodRuntimeMillicores is the CPU weight for the pod runtime cgroup.
	CgroupPodRuntimeMillicores = 1000

	// SelinuxLabelPodRuntime is the SELinux label for workload runtime processes.
	SelinuxLabelPodRuntime = "system_u:system_r:pod_containerd_t:s0"

	// CgroupPodRuntimeReservedMemory is the hard memory protection for runtime processes.
	CgroupPodRuntimeReservedMemory = 196 * 1024 * 1024

	// CgroupDashboardMaxMemory is the hard memory limit for the dashboard process.
	CgroupDashboardMaxMemory = 196 * 1024 * 1024

	// CgroupDashboardMillicores is the CPU weight for the dashboard process.
	CgroupDashboardMillicores = 200

	// FlannelCNI is the string to use Tanos-managed Flannel CNI (default).
	FlannelCNI = "flannel"

	// CustomCNI is the string to use custom CNI managed by Tanos with extra manifests.
	CustomCNI = "custom"

	// NoneCNI is the string to indicate that CNI will not be managed by Talos.
	NoneCNI = "none"

	// CNISELinuxLabel is the SELinux label to be set for CNI configuration overlay mount.
	CNISELinuxLabel = "system_u:object_r:cni_conf_t:s0"

	// DefaultIPv4PodNet is the default IPv4 network range used for workloads.
	DefaultIPv4PodNet = "10.244.0.0/16"

	// DefaultIPv4ServiceNet is the default IPv4 service network range.
	DefaultIPv4ServiceNet = "10.96.0.0/12"

	// DefaultIPv6PodNet is the default IPv6 network range used for workloads.
	DefaultIPv6PodNet = "fc00:db8:10::/56"

	// DefaultIPv6ServiceNet is the default IPv6 service network range.
	DefaultIPv6ServiceNet = "fc00:db8:20::/112"

	// DefaultDNSDomain is the default DNS domain.
	DefaultDNSDomain = "cluster.local"

	// ConfigLoadTimeout is the timeout to wait for the config to be loaded from an external source.
	ConfigLoadTimeout = 3 * time.Hour

	// ConfigLoadAttemptTimeout is the timeout for a single attempt to download config.
	ConfigLoadAttemptTimeout = 3 * time.Minute

	// BootTimeout is the timeout to run all services.
	BootTimeout = 70 * time.Minute

	// FailurePauseTimeout is the timeout for the sequencer failures which can be fixed by updating the machine config.
	FailurePauseTimeout = 35 * time.Minute

	// NodeReadyTimeout is the timeout to wait for the node to be ready (CNI to be running).
	// For bootstrap API, this includes time to run bootstrap.
	NodeReadyTimeout = BootTimeout

	// AnnotationCordonedKey is the annotation key for the nodes cordoned by Talos.
	AnnotationCordonedKey = "talos.dev/cordoned"

	// AnnotationCordonedValue is the annotation key for the nodes cordoned by Talos.
	AnnotationCordonedValue = "true"

	// AnnotationStaticPodSecretsVersion is the annotation key for the static pod secret version.
	AnnotationStaticPodSecretsVersion = "talos.dev/secrets-version"

	// AnnotationStaticPodConfigVersion is the annotation key for the static pod config version.
	AnnotationStaticPodConfigVersion = "talos.dev/config-version"

	// AnnotationStaticPodConfigFileVersion is the annotation key for the static pod configuration file version.
	AnnotationStaticPodConfigFileVersion = "talos.dev/config-file-version"

	// AnnotationOwnedLabels is the annotation key for the list of node labels owned by Talos.
	AnnotationOwnedLabels = "talos.dev/owned-labels"

	// AnnotationOwnedAnnotations is the annotation key for the list of node annotations owned by Talos.
	AnnotationOwnedAnnotations = "talos.dev/owned-annotations"

	// AnnotationOwnedTaints is the annotation key for the list of node taints owned by Talos.
	AnnotationOwnedTaints = "talos.dev/owned-taints"

	// DefaultNTPServer is the NTP server to use if not configured explicitly.
	DefaultNTPServer = "time.cloudflare.com"

	// DefaultPrimaryResolver is the default primary DNS server.
	DefaultPrimaryResolver = "1.1.1.1"

	// DefaultSecondaryResolver is the default secondary DNS server.
	DefaultSecondaryResolver = "8.8.8.8"

	// DefaultClusterIDSize is the default size in bytes for the cluster ID token.
	DefaultClusterIDSize = 32

	// DefaultClusterSecretSize is the default size in bytes for the cluster secret.
	DefaultClusterSecretSize = 32

	// DefaultNodeIdentitySize is the default size in bytes for the node ID.
	DefaultNodeIdentitySize = 32

	// NodeIdentityFilename is the filename to cache node identity across reboots.
	NodeIdentityFilename = "node-identity.yaml"

	// DefaultDiscoveryServiceEndpoint is the default endpoint for Talos discovery service.
	DefaultDiscoveryServiceEndpoint = "https://discovery.talos.dev/"

	// NetworkSelfIPsAnnotation is the node annotation used to list the (comma-separated) IP addresses of the host, as discovered by Talos tooling.
	NetworkSelfIPsAnnotation = "networking.talos.dev/self-ips"

	// NetworkAPIServerPortAnnotation is the node annotation used to report the control plane API server port.
	NetworkAPIServerPortAnnotation = "networking.talos.dev/api-server-port"

	// ClusterNodeIDAnnotation is the node annotation used to represent node ID.
	ClusterNodeIDAnnotation = "cluster.talos.dev/node-id"

	// UdevDir is the path to the udev directory.
	UdevDir = "/usr/lib/udev"

	// UdevRulesPath rules file path.
	UdevRulesPath = UdevDir + "/" + "rules.d/99-talos.rules"

	// UdevRulesLabel rules file SELinux label.
	UdevRulesLabel = "system_u:object_r:udev_rules_t:s0"

	// LoggingFormatJSONLines represents "JSON lines" logging format.
	LoggingFormatJSONLines = "json_lines"

	// SideroLinkName is the interface name for SideroLink.
	SideroLinkName = "siderolink"

	// SideroLinkTunnelName is the tunnel name for SideroLink in tunnel (Wireguard-over-GRPC) mode.
	SideroLinkTunnelName = "siderolinktun"

	// SideroLinkDefaultPeerKeepalive is the interval at which Wireguard Peer Keepalives should be sent.
	SideroLinkDefaultPeerKeepalive = 25 * time.Second

	// PlatformNetworkConfigFilename is the filename to cache platform network configuration reboots.
	PlatformNetworkConfigFilename = "platform-network.yaml"

	// ExtensionServiceConfigPath is the directory path which contains  configuration files of extension services.
	//
	// See pkg/machinery/extensions/services for the file format.
	ExtensionServiceConfigPath = "/usr/local/etc/containers"

	// ExtensionServiceRootfsPath is the path to the extracted rootfs files of extension services.
	ExtensionServiceRootfsPath = "/usr/local/lib/containers"

	// ExtensionServiceUserConfigPath is the path to the user provided extension services config directory.
	ExtensionServiceUserConfigPath = SystemOverlaysPath + "/extensions"

	// DBusServiceSocketPath is the path to the D-Bus socket for the logind mock to connect to.
	DBusServiceSocketPath = SystemRunPath + "/dbus/service.socket"

	// DBusServiceSocketLabel is the SELinux label for the D-Bus socket for the logind mock to connect to.
	DBusServiceSocketLabel = "system_u:object_r:dbus_service_socket_t:s0"

	// DBusClientSocketPath is the path to the D-Bus socket for workload runtime helpers.
	DBusClientSocketPath = "/run/dbus/system_bus_socket"

	// DBusClientSocketLabel is the SELinux label for the D-Bus client socket.
	DBusClientSocketLabel = "system_u:object_r:dbus_client_socket_t:s0"

	// GoVersion is the version of Go compiler this release was built with.
	GoVersion = "go1.25.7"

	// ChuboDir is the default name of the Chubo directory under user home.
	ChuboDir = ".chubo"

	// TalosDir is the legacy name of the Talos directory under user home.
	TalosDir = ".talos"

	// ChuboconfigFilename is the file name of chuboconfig under ChuboDir or under ServiceAccountMountPath inside a pod.
	ChuboconfigFilename = "config"

	// TalosconfigFilename is the legacy file name of talosconfig under TalosDir or under ServiceAccountMountPath inside a pod.
	TalosconfigFilename = "config"

	// ServiceAccountResourceGroup is the group name of the Talos service account CRD.
	ServiceAccountResourceGroup = "talos.dev"

	// ServiceAccountResourceVersion is the version of the Talos service account CRD.
	ServiceAccountResourceVersion = "v1alpha1"

	// ServiceAccountResourceKind is the kind name of the Talos service account CRD.
	ServiceAccountResourceKind = "ServiceAccount"

	// ServiceAccountResourceSingular is the singular name of the Talos service account CRD.
	ServiceAccountResourceSingular = "serviceaccount"

	// ServiceAccountResourceShortName is the short name of the service account CRD.
	ServiceAccountResourceShortName = "tsa"

	// ServiceAccountResourcePlural is the plural name of the service account CRD.
	ServiceAccountResourcePlural = ServiceAccountResourceSingular + "s"

	// ServiceAccountMountPath is the path of the directory in which the Talos service account secrets are mounted.
	ServiceAccountMountPath = "/var/run/secrets/talos.dev"

	// DefaultTrustedRelativeCAFile is the default path to the trusted CA file relative to the /etc.
	DefaultTrustedRelativeCAFile = "ssl/certs/ca-certificates.crt"

	// DefaultTrustedCAFile is the default path to the trusted CA file.
	DefaultTrustedCAFile = "/etc/" + DefaultTrustedRelativeCAFile

	// MachinedMaxProcs is the maximum number of GOMAXPROCS for machined.
	MachinedMaxProcs = 4

	// ApidMaxProcs is the maximum number of GOMAXPROCS for apid.
	ApidMaxProcs = 2

	// TrustdMaxProcs is the maximum number of GOMAXPROCS for trustd.
	TrustdMaxProcs = 2

	// DashboardMaxProcs is the maximum number of GOMAXPROCS for dashboard.
	DashboardMaxProcs = 2

	// APIAuthzRoleMetadataKey is the gRPC metadata key used to submit a role with os:impersonator.
	APIAuthzRoleMetadataKey = "talos-role"

	// KernelLogsTTY is the number of the TTY device (/dev/ttyN) to redirect Kernel logs to.
	KernelLogsTTY = 1

	// DashboardTTY is the number of the TTY device (/dev/ttyN) for dashboard.
	DashboardTTY = 2

	// FlannelVersion is the version of flannel to use.
	//
	// Note: while updating, make sure to copy flannel image from docker.io to ghcr.io:
	//   crane cp docker.io/flannel/flannel:vX.Y.Z ghcr.io/siderolabs/flannel:vX.Y.Z
	//
	// renovate: datasource=github-releases depName=flannel-io/flannel
	FlannelVersion = "v0.27.4"

	// PlatformMetal is the name of the metal platform.
	PlatformMetal = "metal"

	// MetaValuesEnvVar is the name of the environment variable to store encoded meta values for the disk image (installer).
	MetaValuesEnvVar = "INSTALLER_META_BASE64"

	// MaintenanceServiceCommonName is the CN of the maintenance service server certificate.
	MaintenanceServiceCommonName = "maintenance-service.talos.dev"

	// GRPCMaxMessageSize is the maximum message size for Talos API.
	GRPCMaxMessageSize = 32 * 1024 * 1024

	// TalosAPIDefaultCertificateValidityDuration specifies default certificate duration for Talos API generated client certificates.
	TalosAPIDefaultCertificateValidityDuration = time.Hour * 24 * 365

	// DefaultNfTablesTableName is the default name of the nftables table created by Talos.
	DefaultNfTablesTableName = "talos"

	// SystemResolvedPath is the path to the resolved dir.
	SystemResolvedPath = SystemPath + "/resolved"

	// PodResolvConfPath is the path to the pod resolv.conf file.
	PodResolvConfPath = SystemResolvedPath + "/resolv.conf"

	// SyslogListenSocketPath is the path to the syslog socket.
	SyslogListenSocketPath = "/dev/log"

	// ConsoleLogErrorSuppressThreshold is the threshold for suppressing console log errors.
	ConsoleLogErrorSuppressThreshold = 4

	// HostDNSAddress is the address of the host DNS server.
	//
	// Note: 116 = 't' and 108 = 'l' in ASCII.
	HostDNSAddress = "169.254.116.108"

	// MetalAgentModeFlagPath is the path to the file indicating if the node is running in Metal Agent mode.
	MetalAgentModeFlagPath = "/usr/local/etc/is-metal-agent"

	// ImageCachePartitionLabel is the label for the image cache partition.
	ImageCachePartitionLabel = "IMAGECACHE"

	// ImageCacheISOMountPoint is the mount point for the image cache ISO.
	ImageCacheISOMountPoint = "/system/imagecache/iso"

	// ImageCacheDiskMountPoint is the mount point for the image cache partition.
	ImageCacheDiskMountPoint = "/system/imagecache/disk"

	// RegistrydListenAddress is the address to listen on for the registryd service.
	RegistrydListenAddress = "127.0.0.1:3172"

	// UserVolumeMountPoint is the path to the volume mount point for the user volumes.
	UserVolumeMountPoint = "/var/mnt"

	// LogMountPoint is the path to the logs mount point, and ID of the logs volume.
	LogMountPoint = "/var/log"

	// UserVolumePrefix is the prefix for the user volumes.
	UserVolumePrefix = "u-"

	// ExternalVolumePrefix is the prefix for the user volumes.
	ExternalVolumePrefix = "x-"

	// RawVolumePrefix is the prefix for the raw volumes.
	RawVolumePrefix = "r-"

	// ExistingVolumePrefix is the prefix for the existing volumes.
	ExistingVolumePrefix = "e-"

	// SwapVolumePrefix is the prefix for the swap volumes.
	SwapVolumePrefix = "s-"

	// PartitionLabelLength is the length of the partition label.
	//
	// See https://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_entries_(LBA_2%E2%80%9333)
	PartitionLabelLength = 36

	// SPDXPath is the path to the SBOM file(s).
	SPDXPath = "/usr/share/spdx"

	// ExtensionSPDXPath is the path to the SBOM file(s) provided by system extensions.
	ExtensionSPDXPath = "/usr/local/share/spdx"

	// EncryptionSaltFilename is the filename for the encryption salt file.
	EncryptionSaltFilename = "encryption-salt.yaml"

	// DiskEncryptionSaltSize is the size of the disk encryption salt in bytes.
	DiskEncryptionSaltSize = 32

	// SideroV1KeysDirEnvVar is the environment variable that points to the directory containing user PGP keys for SideroV1 auth.
	SideroV1KeysDirEnvVar = "SIDEROV1_KEYS_DIR"

	// SideroV1KeysDir is the default directory containing user PGP keys for SideroV1 auth.
	SideroV1KeysDir = "keys"

	// ContainerMarkerFilePath is the path to the file added to container builds of Talos for platform detection.
	ContainerMarkerFilePath = "/usr/etc/in-container"

	// DefaultOOMTriggerExpression is the default CEL expression used to determine whether to trigger OOM.
	DefaultOOMTriggerExpression = `(multiply_qos_vectors(d_qos_memory_full_total, {System: 8.0, Podruntime: 4.0}) > 3000.0 &&
	     multiply_qos_vectors(qos_memory_full_avg10, {System: 1.0, Podruntime: 1.0}) > 5.0) ||
		(memory_full_avg10 > 75.0 && time_since_trigger > duration("10s"))`

	// DefaultOOMCgroupRankingExpression is the default CEL expression used to rank cgroups for OOM killer.
	DefaultOOMCgroupRankingExpression = `memory_max.hasValue() ? 0.0 :
		{Besteffort: 1.0, Burstable: 0.5, Guaranteed: 0.0, Podruntime: 0.0, System: 0.0}[class] *
		   double(memory_current.orValue(0u))`

	// OOMActionLogKeep is the number of OOM action log entries to keep in memory.
	OOMActionLogKeep = 50

	// SDStubCmdlineExtraOEMVar is the name of the SMBIOS OEM variable that can be used to pass extra kernel command line parameters to systemd-stub.
	SDStubCmdlineExtraOEMVar = "io.systemd.stub.kernel-cmdline-extra"

	// LogRotateThreshold is the size (in bytes), upon exceeding which the log file should be rotated.
	LogRotateThreshold = 5 * 1024 * 1024

	// LogFlushPeriod is the period for flushing in-memory log buffers to the filesystem.
	LogFlushPeriod = 15 * time.Second
)

// names of variable that can be substituted in the talos.config kernel parameter.
const (
	UUIDKey         = "uuid"
	SerialNumberKey = "serial"
	HostnameKey     = "hostname"
	MacKey          = "mac"
	CodeKey         = "code"
)

// SELinuxLabeledPath is an object used to describe overlay mounts with SELinux labels applied on creation.
type SELinuxLabeledPath struct {
	Path  string
	Label string
}

// Overlays is the set of paths to create overlay mounts for.
var Overlays = []SELinuxLabeledPath{
	{"/etc/cni", CNISELinuxLabel},
	{"/opt", OptSELinuxLabel},
}

// DefaultDroppedCapabilities is the default set of capabilities to drop.
var DefaultDroppedCapabilities = map[string]struct{}{
	"cap_sys_boot":   {},
	"cap_sys_module": {},
}

// UdevdDroppedCapabilities is the set of capabilities to drop for udevd.
var UdevdDroppedCapabilities = map[string]struct{}{
	"cap_sys_boot": {},
}

// ValidEffects is the set of valid taint effects.
var ValidEffects = []string{
	"NoSchedule",
	"PreferNoSchedule",
	"NoExecute",
}

// OSReleaseTemplate is the template for /etc/os-release.
const OSReleaseTemplate = `NAME="%[1]s"
ID=%[2]s
VERSION_ID=%[3]s
PRETTY_NAME="%[1]s (%[3]s)"
HOME_URL="https://www.talos.dev/"
BUG_REPORT_URL="https://github.com/chubo-dev/chubo/issues"
VENDOR_NAME="Sidero Labs"
VENDOR_URL="https://www.siderolabs.com/"
`
