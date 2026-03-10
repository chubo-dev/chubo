---
description: Config defines the v1alpha1.Config machine configuration document.
title: Config
---

<!-- markdownlint-disable -->









{{< highlight yaml >}}
version: v1alpha1
machine: # ...
cluster: # ...
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`version` |string |Indicates the schema used to decode the contents.  |`v1alpha1`<br /> |
|`debug` |bool |Enable verbose logging to the console.<br>All system containers logs will flow into serial console.<br><br>**Note:** To avoid breaking bootstrap flow enable this option only if serial console can handle high message throughput.  |`true`<br />`yes`<br />`false`<br />`no`<br /> |
|`machine` |<a href="#Config.machine">MachineConfig</a> |Provides machine specific configuration options.  | |
|`cluster` |<a href="#Config.cluster">ClusterConfig</a> |Provides cluster specific configuration options.  | |




## machine {#Config.machine}

MachineConfig represents the machine-specific config values.



{{< highlight yaml >}}
machine:
    type: controlplane
    # InstallConfig represents the installation options for preparing a node.
    install:
        disk: /dev/sda # The disk used for installations.
        image: ghcr.io/siderolabs/installer:latest # Allows for supplying the image used to perform the installation.
        wipe: false # Indicates if the installation disk should be wiped at installation time.
        grubUseUKICmdline: true # Indicates if legacy GRUB bootloader should use kernel cmdline from the UKI instead of building it on the host.

        # # Look up disk using disk attributes like model, size, serial and others.
        # diskSelector:
        #     size: 4GB # Disk size.
        #     model: WDC* # Disk model `/sys/block/<dev>/device/model`.
        #     busPath: /pci0000:00/0000:00:17.0/ata1/host0/target0:0:0/0:0:0:0 # Disk bus path.
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`type` |string |Defines the role of the machine within the cluster.<br><br>**Control Plane**<br><br>Control Plane node type designates the node as a control plane member.<br>This means it hosts the core cluster management services.<br><br>**Worker**<br><br>Worker node type designates the node as a worker node.<br>This means it will be an available compute node for scheduling workloads.<br><br>This node type was previously known as "join"; that value is still supported but deprecated.  |`controlplane`<br />`worker`<br /> |
|`token` |string |The `token` is used by a machine to join the PKI of the cluster.<br>Using this token, a machine will create a certificate signing request (CSR), and request a certificate that will be used as its' identity. <details><summary>Show example(s)</summary>example token:{{< highlight yaml >}}
token: 328hom.uqjzh6jnn2eie9oi
{{< /highlight >}}</details> | |
|`ca` |PEMEncodedCertificateAndKey |The root certificate authority of the PKI.<br>It is composed of a base64 encoded `crt` and `key`. <details><summary>Show example(s)</summary>machine CA example:{{< highlight yaml >}}
ca:
    crt: LS0tIEVYQU1QTEUgQ0VSVElGSUNBVEUgLS0t
    key: LS0tIEVYQU1QTEUgS0VZIC0tLQ==
{{< /highlight >}}</details> | |
|`acceptedCAs` |[]PEMEncodedCertificate |The certificates issued by certificate authorities are accepted in addition to issuing 'ca'.<br>It is composed of a base64 encoded `crt``.  | |
|`certSANs` |[]string |Extra certificate subject alternative names for the machine's certificate.<br>By default, all non-loopback interface IPs are automatically added to the certificate's SANs. <details><summary>Show example(s)</summary>Uncomment this to enable SANs.:{{< highlight yaml >}}
certSANs:
    - 10.0.0.10
    - 172.16.0.10
    - 192.168.0.10
{{< /highlight >}}</details> | |
|`install` |<a href="#Config.machine.install">InstallConfig</a> |Used to provide instructions for installations.<br><br>Note that this configuration section gets silently ignored by images that are considered pre-installed.<br>To make sure installation uses this configuration, boot from ISO or PXE. <details><summary>Show example(s)</summary>MachineInstall config usage example.:{{< highlight yaml >}}
install:
    disk: /dev/sda # The disk used for installations.
    image: ghcr.io/siderolabs/installer:latest # Allows for supplying the image used to perform the installation.
    wipe: false # Indicates if the installation disk should be wiped at installation time.
    grubUseUKICmdline: true # Indicates if legacy GRUB bootloader should use kernel cmdline from the UKI instead of building it on the host.

    # # Look up disk using disk attributes like model, size, serial and others.
    # diskSelector:
    #     size: 4GB # Disk size.
    #     model: WDC* # Disk model `/sys/block/<dev>/device/model`.
    #     busPath: /pci0000:00/0000:00:17.0/ata1/host0/target0:0:0/0:0:0:0 # Disk bus path.
{{< /highlight >}}</details> | |
|`files` |<a href="#Config.machine.files.">[]MachineFile</a> |Allows the addition of user specified files.<br>The value of `op` can be `create`, `overwrite`, or `append`.<br>In the case of `create`, `path` must not exist.<br>In the case of `overwrite`, and `append`, `path` must be a valid file.<br>If an `op` value of `append` is used, the existing file will be appended.<br>Note that the file contents are not required to be base64 encoded. <details><summary>Show example(s)</summary>MachineFiles usage example.:{{< highlight yaml >}}
files:
    - content: '...' # The contents of the file.
      permissions: 0o666 # The file's permissions in octal.
      path: /tmp/file.txt # The path of the file.
      op: append # The operation to use
{{< /highlight >}}</details> | |
|`sysctls` |map[string]string |Used to configure the machine's sysctls. <details><summary>Show example(s)</summary>MachineSysctls usage example.:{{< highlight yaml >}}
sysctls:
    kernel.domainname: chubo.dev
    net.ipv4.ip_forward: "0"
    net/ipv6/conf/eth0.100/disable_ipv6: "1"
{{< /highlight >}}</details> | |
|`sysfs` |map[string]string |Used to configure the machine's sysfs. <details><summary>Show example(s)</summary>MachineSysfs usage example.:{{< highlight yaml >}}
sysfs:
    devices.system.cpu.cpu0.cpufreq.scaling_governor: performance
{{< /highlight >}}</details> | |
|`features` |<a href="#Config.machine.features">FeaturesConfig</a> |Features describe individual OS features that can be switched on or off. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
features:
    diskQuotaSupport: true # Enable XFS project quota support for EPHEMERAL partition and user disks.
{{< /highlight >}}</details> | |
|`udev` |<a href="#Config.machine.udev">UdevConfig</a> |Configures the udev system. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
udev:
    # List of udev rules to apply to the udev system
    rules:
        - SUBSYSTEM=="drm", KERNEL=="renderD*", GROUP="44", MODE="0660"
{{< /highlight >}}</details> | |
|`logging` |<a href="#Config.machine.logging">LoggingConfig</a> |Configures the logging system. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
logging:
    # Logging destination.
    destinations:
        - endpoint: tcp://1.2.3.4:12345 # Where to send logs. Supported protocols are "tcp" and "udp".
          format: json_lines # Logs format.
{{< /highlight >}}</details> | |
|`kernel` |<a href="#Config.machine.kernel">KernelConfig</a> |Configures the kernel. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
kernel:
    # Kernel modules to load.
    modules:
        - name: btrfs # Module name.
{{< /highlight >}}</details> | |
|`seccompProfiles` |<a href="#Config.machine.seccompProfiles.">[]MachineSeccompProfile</a> |Configures the seccomp profiles for the machine. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
seccompProfiles:
    - name: audit.json # The `name` field is used to provide the file name of the seccomp profile.
      # The `value` field is used to provide the seccomp profile.
      value:
        defaultAction: SCMP_ACT_LOG
{{< /highlight >}}</details> | |
|`baseRuntimeSpecOverrides` |Unstructured |Override (patch) settings in the default OCI runtime spec for CRI containers.<br><br>It can be used to set default container settings that are not configurable at the scheduler layer,<br>for example default ulimits.<br>Note: this change applies to all newly created containers, and it requires a reboot to take effect. <details><summary>Show example(s)</summary>override default open file limit:{{< highlight yaml >}}
baseRuntimeSpecOverrides:
    process:
        rlimits:
            - hard: 1024
              soft: 1024
              type: RLIMIT_NOFILE
{{< /highlight >}}</details> | |




### install {#Config.machine.install}

InstallConfig represents the installation options for preparing a node.



{{< highlight yaml >}}
machine:
    install:
        disk: /dev/sda # The disk used for installations.
        image: ghcr.io/siderolabs/installer:latest # Allows for supplying the image used to perform the installation.
        wipe: false # Indicates if the installation disk should be wiped at installation time.
        grubUseUKICmdline: true # Indicates if legacy GRUB bootloader should use kernel cmdline from the UKI instead of building it on the host.

        # # Look up disk using disk attributes like model, size, serial and others.
        # diskSelector:
        #     size: 4GB # Disk size.
        #     model: WDC* # Disk model `/sys/block/<dev>/device/model`.
        #     busPath: /pci0000:00/0000:00:17.0/ata1/host0/target0:0:0/0:0:0:0 # Disk bus path.
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`disk` |string |The disk used for installations. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
disk: /dev/sda
{{< /highlight >}}{{< highlight yaml >}}
disk: /dev/nvme0
{{< /highlight >}}</details> | |
|`diskSelector` |<a href="#Config.machine.install.diskSelector">InstallDiskSelector</a> |Look up disk using disk attributes like model, size, serial and others.<br>Always has priority over `disk`. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
diskSelector:
    size: '>= 1TB' # Disk size.
    model: WDC* # Disk model `/sys/block/<dev>/device/model`.

    # # Disk bus path.
    # busPath: /pci0000:00/0000:00:17.0/ata1/host0/target0:0:0/0:0:0:0
    # busPath: /pci0000:00/*
{{< /highlight >}}</details> | |
|`image` |string |Allows for supplying the image used to perform the installation.<br>Image reference for each Chubo OS release can be found on<br>[GitHub releases page](https://github.com/chubo-dev/chubo/releases). <details><summary>Show example(s)</summary>{{< highlight yaml >}}
image: ghcr.io/siderolabs/installer:latest
{{< /highlight >}}</details> | |
|`wipe` |bool |Indicates if the installation disk should be wiped at installation time.<br>Defaults to `true`.  |`true`<br />`yes`<br />`false`<br />`no`<br /> |
|`legacyBIOSSupport` |bool |Indicates if MBR partition should be marked as bootable (active).<br>Should be enabled only for the systems with legacy BIOS that doesn't support GPT partitioning scheme.  | |
|`grubUseUKICmdline` |bool |Indicates if legacy GRUB bootloader should use kernel cmdline from the UKI instead of building it on the host.<br>This changes the way cmdline is managed with GRUB bootloader to be more consistent with UKI/systemd-boot.  | |




#### diskSelector {#Config.machine.install.diskSelector}

InstallDiskSelector represents a disk query parameters for the install disk lookup.



{{< highlight yaml >}}
machine:
    install:
        diskSelector:
            size: '>= 1TB' # Disk size.
            model: WDC* # Disk model `/sys/block/<dev>/device/model`.

            # # Disk bus path.
            # busPath: /pci0000:00/0000:00:17.0/ata1/host0/target0:0:0/0:0:0:0
            # busPath: /pci0000:00/*
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`size` |InstallDiskSizeMatcher |Disk size. <details><summary>Show example(s)</summary>Select a disk which size is equal to 4GB.:{{< highlight yaml >}}
size: 4GB
{{< /highlight >}}Select a disk which size is greater than 1TB.:{{< highlight yaml >}}
size: '> 1TB'
{{< /highlight >}}Select a disk which size is less or equal than 2TB.:{{< highlight yaml >}}
size: <= 2TB
{{< /highlight >}}</details> | |
|`name` |string |Disk name `/sys/block/<dev>/device/name`.  | |
|`model` |string |Disk model `/sys/block/<dev>/device/model`.  | |
|`serial` |string |Disk serial number `/sys/block/<dev>/serial`.  | |
|`modalias` |string |Disk modalias `/sys/block/<dev>/device/modalias`.  | |
|`uuid` |string |Disk UUID `/sys/block/<dev>/uuid`.  | |
|`wwid` |string |Disk WWID `/sys/block/<dev>/wwid`.  | |
|`type` |InstallDiskType |Disk Type.  |`ssd`<br />`hdd`<br />`nvme`<br />`sd`<br /> |
|`busPath` |string |Disk bus path. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
busPath: /pci0000:00/0000:00:17.0/ata1/host0/target0:0:0/0:0:0:0
{{< /highlight >}}{{< highlight yaml >}}
busPath: /pci0000:00/*
{{< /highlight >}}</details> | |








### files[] {#Config.machine.files.}

MachineFile represents a file to write to disk.



{{< highlight yaml >}}
machine:
    files:
        - content: '...' # The contents of the file.
          permissions: 0o666 # The file's permissions in octal.
          path: /tmp/file.txt # The path of the file.
          op: append # The operation to use
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`content` |string |The contents of the file.  | |
|`permissions` |FileMode |The file's permissions in octal.  | |
|`path` |string |The path of the file.  | |
|`op` |string |The operation to use  |`create`<br />`append`<br />`overwrite`<br /> |






### features {#Config.machine.features}

FeaturesConfig describes individual OS features that can be switched on or off.



{{< highlight yaml >}}
machine:
    features:
        diskQuotaSupport: true # Enable XFS project quota support for EPHEMERAL partition and user disks.
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`diskQuotaSupport` |bool |Enable XFS project quota support for EPHEMERAL partition and user disks.  | |
|`hostDNS` |<a href="#Config.machine.features.hostDNS">HostDNSConfig</a> |Configures host DNS caching resolver.  | |
|`imageCache` |<a href="#Config.machine.features.imageCache">ImageCacheConfig</a> |Enable Image Cache feature.  | |
|`nodeAddressSortAlgorithm` |string |Select the node address sort algorithm.<br>The 'v1' algorithm sorts addresses by the address itself.<br>The 'v2' algorithm prefers more specific prefixes.<br>If unset, defaults to 'v1'.  | |




#### hostDNS {#Config.machine.features.hostDNS}

HostDNSConfig describes the configuration for the host DNS resolver.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`enabled` |bool |Enable host DNS caching resolver.  | |
|`resolveMemberNames` |bool |Resolve member hostnames using the host DNS resolver.<br><br>When enabled, cluster member hostnames and node names are resolved using the host DNS resolver.<br>This requires service discovery to be enabled.  | |






#### imageCache {#Config.machine.features.imageCache}

ImageCacheConfig describes the configuration for the Image Cache feature.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`localEnabled` |bool |Enable local image cache.  | |








### udev {#Config.machine.udev}

UdevConfig describes how the udev system should be configured.



{{< highlight yaml >}}
machine:
    udev:
        # List of udev rules to apply to the udev system
        rules:
            - SUBSYSTEM=="drm", KERNEL=="renderD*", GROUP="44", MODE="0660"
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`rules` |[]string |List of udev rules to apply to the udev system  | |






### logging {#Config.machine.logging}

LoggingConfig struct configures OS logging.



{{< highlight yaml >}}
machine:
    logging:
        # Logging destination.
        destinations:
            - endpoint: tcp://1.2.3.4:12345 # Where to send logs. Supported protocols are "tcp" and "udp".
              format: json_lines # Logs format.
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`destinations` |<a href="#Config.machine.logging.destinations.">[]LoggingDestination</a> |Logging destination.  | |




#### destinations[] {#Config.machine.logging.destinations.}

LoggingDestination struct configures logging destinations.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`endpoint` |<a href="#Config.machine.logging.destinations..endpoint">Endpoint</a> |Where to send logs. Supported protocols are "tcp" and "udp". <details><summary>Show example(s)</summary>{{< highlight yaml >}}
endpoint: udp://127.0.0.1:12345
{{< /highlight >}}{{< highlight yaml >}}
endpoint: tcp://1.2.3.4:12345
{{< /highlight >}}</details> | |
|`format` |string |Logs format.  |`json_lines`<br /> |
|`extraTags` |map[string]string |Extra tags (key-value) pairs to attach to every log message sent.  | |




##### endpoint {#Config.machine.logging.destinations..endpoint}

Endpoint represents the endpoint URL parsed out of the machine config.



{{< highlight yaml >}}
machine:
    logging:
        destinations:
            - endpoint: https://1.2.3.4:6443
{{< /highlight >}}

{{< highlight yaml >}}
machine:
    logging:
        destinations:
            - endpoint: https://cluster1.internal:6443
{{< /highlight >}}

{{< highlight yaml >}}
machine:
    logging:
        destinations:
            - endpoint: udp://127.0.0.1:12345
{{< /highlight >}}

{{< highlight yaml >}}
machine:
    logging:
        destinations:
            - endpoint: tcp://1.2.3.4:12345
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|










### kernel {#Config.machine.kernel}

KernelConfig struct configures the Linux kernel.



{{< highlight yaml >}}
machine:
    kernel:
        # Kernel modules to load.
        modules:
            - name: btrfs # Module name.
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`modules` |<a href="#Config.machine.kernel.modules.">[]KernelModuleConfig</a> |Kernel modules to load.  | |




#### modules[] {#Config.machine.kernel.modules.}

KernelModuleConfig struct configures Linux kernel modules to load.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`name` |string |Module name.  | |
|`parameters` |[]string |Module parameters, changes applied after reboot.  | |








### seccompProfiles[] {#Config.machine.seccompProfiles.}

MachineSeccompProfile defines seccomp profiles for the machine.



{{< highlight yaml >}}
machine:
    seccompProfiles:
        - name: audit.json # The `name` field is used to provide the file name of the seccomp profile.
          # The `value` field is used to provide the seccomp profile.
          value:
            defaultAction: SCMP_ACT_LOG
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`name` |string |The `name` field is used to provide the file name of the seccomp profile.  | |
|`value` |Unstructured |The `value` field is used to provide the seccomp profile.  | |








## cluster {#Config.cluster}

ClusterConfig represents cluster identity and discovery settings.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`id` |string |Unique cluster ID used for node membership and discovery.  | |
|`secret` |string |Shared cluster secret used to authenticate and protect membership discovery data.  | |
|`clusterName` |string |Human-friendly cluster name.  | |
|`discovery` |<a href="#Config.cluster.discovery">ClusterDiscoveryConfig</a> |Cluster membership discovery settings.  | |




### discovery {#Config.cluster.discovery}

ClusterDiscoveryConfig struct configures cluster membership discovery.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`enabled` |bool |Enable the cluster membership discovery feature.<br>Cluster discovery is based on individual registries which are configured under the registries field.  | |
|`registries` |<a href="#Config.cluster.discovery.registries">DiscoveryRegistriesConfig</a> |Configure registries used for cluster member discovery.  | |




#### registries {#Config.cluster.discovery.registries}

DiscoveryRegistriesConfig struct configures cluster membership discovery.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`service` |<a href="#Config.cluster.discovery.registries.service">RegistryServiceConfig</a> |Service registry is using an external service to push and pull information about cluster members.  | |




##### service {#Config.cluster.discovery.registries.service}

RegistryServiceConfig struct configures service discovery registry.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`disabled` |bool |Disable external service discovery registry.  | |
|`endpoint` |string |External service endpoint. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
endpoint: https://discovery.chubo.dev/
{{< /highlight >}}</details> | |














