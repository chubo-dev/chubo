---
description: Talosctl CLI tool reference.
title: CLI
---

<!-- markdownlint-disable -->

## chuboctl apply-config

Apply a new configuration to a node

```
chuboctl apply-config [flags]
```

### Options

```
      --cert-fingerprint strings                    list of server certificate fingeprints to accept (defaults to no check)
      --chuboconfig string                          The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string                              Cluster to connect to if a proxy endpoint is used.
  -p, --config-patch stringArray                    the list of config patches to apply to the local config file before sending it to the node
      --context string                              Context to be used in command
      --dry-run                                     check how the config change will be applied in dry-run mode
  -e, --endpoints strings                           override default endpoints in client configuration
  -f, --file string                                 the filename of the updated configuration
  -h, --help                                        help for apply-config
  -i, --insecure                                    apply the config using the insecure (encrypted with no auth) maintenance service
  -m, --mode auto, no-reboot, reboot, staged, try   apply config mode (default auto)
  -n, --nodes strings                               target the specified nodes
      --siderov1-keys-dir string                    The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string                          Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --timeout duration                            the config will be rolled back after specified timeout (if try mode is selected) (default 1m0s)
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl cgroups

Retrieve cgroups usage information

### Synopsis

The cgroups command fetches control group v2 (cgroupv2) usage details from the machine.
Several presets are available to focus on specific cgroup subsystems:

* cpu
* cpuset
* io
* memory
* process
* swap

You can specify the preset using the --preset flag.

Alternatively, a custom schema can be provided using the --schema-file flag.
To see schema examples, refer to https://github.com/chubo-dev/chubo/tree/main/cmd/talosctl/cmd/talos/cgroupsprinter/schemas.


```
chuboctl cgroups [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for cgroups
  -n, --nodes strings              target the specified nodes
      --preset string              preset name (one of: [cpu cpuset io memory process psi swap])
      --schema-file string         path to the columns schema file
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --skip-cri-resolve           do not resolve cgroup names via a request to CRI
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl cluster create dev

Creates a local QEMU-based cluster for Talos development.

```
chuboctl cluster create dev [flags]
```

### Options

```
      --arch string                              cluster architecture (default "arm64")
      --cidr string                              CIDR of the cluster network (IPv4, ULA network for IPv6 is derived in automated way) (default "10.5.0.0/24")
      --config-patch stringArray                 patch generated machineconfigs (applied to all node types), use @file to read a patch from file
      --config-patch-control-plane stringArray   patch generated machineconfigs (applied to 'controlplane' type)
      --config-patch-worker stringArray          patch generated machineconfigs (applied to 'worker' type)
      --control-plane-port int                   control plane port (load balancer and local API port) (default 6443)
      --controlplanes int                        the number of controlplanes to create (default 1)
      --cpus string                              the share of CPUs as fraction for each control plane/VM (default "2.0")
      --cpus-workers string                      the share of CPUs as fraction for each worker/VM (default "2.0")
      --custom-cni-url string                    install custom CNI from the URL (Talos cluster)
      --disable-dhcp-hostname                    skip announcing hostname via DHCP
      --disk int                                 default limit on disk size in MB (each VM) (default 6144)
      --disk-block-size uint                     disk block size (default 512)
      --disk-encryption-key-types stringArray    encryption key types to use for disk encryption (uuid, kms) (default [uuid])
      --disk-image-path string                   disk image to use
      --disk-preallocate                         whether disk space should be preallocated (default true)
      --dns-domain string                        the dns domain to use for cluster (default "cluster.local")
      --encrypt-ephemeral                        enable ephemeral partition encryption
      --encrypt-state                            enable state partition encryption
      --encrypt-user-volumes                     enable ephemeral partition encryption
      --endpoint string                          use endpoint instead of provider defaults
      --extra-boot-kernel-args string            add extra kernel args to the initial boot from vmlinuz and initramfs
      --extra-disks int                          number of extra disks to create for each worker VM
      --extra-disks-drivers strings              driver for each extra disk (virtio, ide, ahci, scsi, nvme, megaraid)
      --extra-disks-serials strings              serials for each extra disk
      --extra-disks-size int                     default limit on disk size in MB (each VM) (default 5120)
      --extra-disks-tags strings                 tags for each extra disk (only used by virtiofs)
      --extra-uefi-search-paths strings          additional search paths for UEFI firmware (only applies when UEFI is enabled)
  -h, --help                                     help for dev
      --image-cache-path string                  path to image cache
      --image-cache-port uint16                  port on which to serve image cache (default 5000)
      --image-cache-tls-cert-file string         path to image cache TLS cert
      --image-cache-tls-key-file string          path to image cache TLS key
      --init-node-as-endpoint                    use init node as endpoint instead of any load balancer endpoint
      --initrd-path string                       initramfs image to use (default "_out/initramfs-${ARCH}.xz")
      --install-image string                     the installer image to use (default "ghcr.io/siderolabs/installer:v1.13.0-alpha.1-240-g3ac26b8f9")
      --ipv4                                     enable IPv4 network in the cluster (default true)
      --ipv6                                     enable IPv6 network in the cluster
      --ipxe-boot-script string                  iPXE boot script (URL) to use
      --iso-path string                          the ISO path to use for the initial boot
      --memory string(mb,gb)                     the limit on memory usage for each control plane/VM (default 2.0GiB)
      --memory-workers string(mb,gb)             the limit on memory usage for each worker/VM (default 2.0GiB)
      --mtu int                                  MTU of the cluster network (default 1500)
      --nameservers strings                      list of nameservers to use (default [8.8.8.8,1.1.1.1,2001:4860:4860::8888,2606:4700:4700::1111])
      --omni-api-endpoint string                 the Omni API endpoint (must include a scheme, a hostname and a join token, e.g. 'https://siderolink.omni.example?jointoken=foobar')
      --registry-insecure-skip-verify strings    list of registry hostnames to skip TLS verification for
      --registry-mirror strings                  list of registry mirrors to use in format: <registry host>=<mirror URL>
      --skip-injecting-config                    skip injecting config from embedded metadata server, write config files to current directory
      --skip-injecting-extra-cmdline             skip injecting extra kernel cmdline parameters via EFI vars through bootloader
      --talos-version string                     the desired Talos version to generate config for (default "v1.13.0-alpha.1-240-g3ac26b8f9")
      --talosconfig string                       The location to save the generated client configuration file to. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --uki-path string                          the UKI image path to use for the initial boot
      --usb-path string                          the USB stick image path to use for the initial boot
      --use-vip                                  use a virtual IP for the controlplane endpoint instead of the loadbalancer
      --user-volumes strings                     list of user volumes to create for each VM in format: <name1>:<size1>:<name2>:<size2>
      --vmlinuz-path string                      the compressed kernel image to use (default "_out/vmlinuz-${ARCH}")
      --wait                                     wait for the cluster to be ready before returning (default true)
      --wait-timeout duration                    timeout to wait for the cluster to be ready (default 20m0s)
      --wireguard-cidr string                    CIDR of the wireguard network
      --with-apply-config                        enable apply config when the VM is starting in maintenance mode
      --with-bootloader                          enable bootloader to load kernel and initramfs from disk image after install (default true)
      --with-cluster-discovery                   enable cluster discovery (default true)
      --with-debug                               enable debug in Talos config to send service logs to the console
      --with-firewall string                     inject firewall rules into the cluster, value is default policy - accept/block
      --with-init-node                           create the cluster with an init node
      --with-iommu                               enable IOMMU support, this also add a new PCI root port and an interface attached to it
      --with-json-logs                           enable JSON logs receiver and configure Talos to send logs there
      --with-siderolink true                     enables the use of siderolink agent as configuration apply mechanism. true or `wireguard` enables the agent, `tunnel` enables the agent with grpc tunneling (default none)
      --with-tpm1_2                              enable TPM 1.2 emulation support using swtpm
      --with-tpm2                                enable TPM 2.0 emulation support using swtpm
      --with-uefi                                enable UEFI on x86_64 architecture (default true)
      --with-uuid-hostnames                      use machine UUIDs as default hostnames
      --workers int                              the number of workers to create (default 1)
```

### Options inherited from parent commands

```
      --name string    the name of the cluster (default "chubo-default")
      --state string   directory path to store cluster state (default "/Users/francesco/.chubo/clusters")
```

### SEE ALSO

* [chuboctl cluster create](#chuboctl-cluster-create)	 - Create a local Talos cluster.

## chuboctl cluster create docker

Create a local Docker based cluster

```
chuboctl cluster create docker [flags]
```

### Options

```
      --config-patch stringArray                 patch generated machineconfigs (applied to all node types), use @file to read a patch from file
      --config-patch-controlplanes stringArray   patch generated machineconfigs (applied to 'controlplane' type)
      --config-patch-workers stringArray         patch generated machineconfigs (applied to 'worker' type)
      --cpus-controlplanes string                the share of CPUs as fraction for each control plane/VM (default "2.0")
      --cpus-workers string                      the share of CPUs as fraction for each worker/VM (default "2.0")
  -p, --exposed-ports string                     comma-separated list of ports/protocols to expose on init node. Ex -p <hostPort>:<containerPort>/<protocol (tcp or udp)>
  -h, --help                                     help for docker
      --host-ip string                           Host IP to forward exposed ports to (default "0.0.0.0")
      --image string                             the talos image to run (default "ghcr.io/siderolabs/talos:v1.13.0-alpha.1-240-g3ac26b8f9")
      --memory-controlplanes string(mb,gb)       the limit on memory usage for each control plane/VM (default 2.0GiB)
      --memory-workers string(mb,gb)             the limit on memory usage for each worker/VM (default 2.0GiB)
      --mount mount                              attach a mount to the container (docker --mount syntax)
      --subnet string                            Docker network subnet CIDR (default "10.5.0.0/24")
      --talosconfig-destination string           The location to save the generated client configuration file to. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --workers int                              the number of workers to create (default 1)
```

### Options inherited from parent commands

```
      --name string    the name of the cluster (default "chubo-default")
      --state string   directory path to store cluster state (default "/Users/francesco/.chubo/clusters")
```

### SEE ALSO

* [chuboctl cluster create](#chuboctl-cluster-create)	 - Create a local Talos cluster.

## chuboctl cluster create qemu

Create a local QEMU based Talos cluster.

### Synopsis

Create a local QEMU based Talos cluster.

Available presets:
  - iso: Configure Talos to boot from an ISO from the Image Factory.
  - iso-secureboot: Configure Talos for Secureboot via ISO. Only available on Linux hosts.
  - pxe: Configure Talos to boot via PXE from the Image Factory.
  - disk-image: Configure Talos to boot from a disk image from the Image Factory.
  - maintenance: Skip applying machine configuration and leave the machines in maintenance mode. The machine configuration files are written to the working directory.

Note: exactly one of 'iso', 'iso-secureboot', 'pxe' or 'disk-image' presets must be specified.


```
chuboctl cluster create qemu [flags]
```

### Options

```
      --cidr string                              CIDR of the cluster network (default "10.5.0.0/24")
      --config-patch stringArray                 patch generated machineconfigs (applied to all node types), use @file to read a patch from file
      --config-patch-controlplanes stringArray   patch generated machineconfigs (applied to 'controlplane' type)
      --config-patch-workers stringArray         patch generated machineconfigs (applied to 'worker' type)
      --controlplanes int                        the number of controlplanes to create (default 1)
      --cpus-controlplanes string                the share of CPUs as fraction for each control plane/VM (default "2.0")
      --cpus-workers string                      the share of CPUs as fraction for each worker/VM (default "2.0")
      --disks disks                              list of disks to create in format "<driver1>:<size1>" (disks after the first one are added only to worker machines) (default virtio:10GiB,virtio:6GiB)
  -h, --help                                     help for qemu
      --image-factory-url string                 image factory url (default "https://factory.talos.dev/")
      --memory-controlplanes string(mb,gb)       the limit on memory usage for each control plane/VM (default 2.0GiB)
      --memory-workers string(mb,gb)             the limit on memory usage for each worker/VM (default 2.0GiB)
      --omni-api-endpoint string                 the Omni API endpoint (must include a scheme, a hostname and a join token, e.g. 'https://siderolink.omni.example?jointoken=foobar')
      --presets strings                          list of presets to apply (default [iso])
      --schematic-id string                      image factory schematic id (defaults to an empty schematic)
      --talos-version string                     the desired talos version (default "v1.13.0-alpha.1-240-g3ac26b8f9")
      --talosconfig-destination string           The location to save the generated client configuration file to. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --workers int                              the number of workers to create (default 1)
```

### Options inherited from parent commands

```
      --name string    the name of the cluster (default "chubo-default")
      --state string   directory path to store cluster state (default "/Users/francesco/.chubo/clusters")
```

### SEE ALSO

* [chuboctl cluster create](#chuboctl-cluster-create)	 - Create a local Talos cluster.

## chuboctl cluster create dev

Creates a local QEMU-based cluster for Talos development.

```
chuboctl cluster create dev [flags]
```

### Options

```
      --arch string                              cluster architecture (default "arm64")
      --cidr string                              CIDR of the cluster network (IPv4, ULA network for IPv6 is derived in automated way) (default "10.5.0.0/24")
      --config-patch stringArray                 patch generated machineconfigs (applied to all node types), use @file to read a patch from file
      --config-patch-control-plane stringArray   patch generated machineconfigs (applied to 'controlplane' type)
      --config-patch-worker stringArray          patch generated machineconfigs (applied to 'worker' type)
      --control-plane-port int                   control plane port (load balancer and local API port) (default 6443)
      --controlplanes int                        the number of controlplanes to create (default 1)
      --cpus string                              the share of CPUs as fraction for each control plane/VM (default "2.0")
      --cpus-workers string                      the share of CPUs as fraction for each worker/VM (default "2.0")
      --custom-cni-url string                    install custom CNI from the URL (Talos cluster)
      --disable-dhcp-hostname                    skip announcing hostname via DHCP
      --disk int                                 default limit on disk size in MB (each VM) (default 6144)
      --disk-block-size uint                     disk block size (default 512)
      --disk-encryption-key-types stringArray    encryption key types to use for disk encryption (uuid, kms) (default [uuid])
      --disk-image-path string                   disk image to use
      --disk-preallocate                         whether disk space should be preallocated (default true)
      --dns-domain string                        the dns domain to use for cluster (default "cluster.local")
      --encrypt-ephemeral                        enable ephemeral partition encryption
      --encrypt-state                            enable state partition encryption
      --encrypt-user-volumes                     enable ephemeral partition encryption
      --endpoint string                          use endpoint instead of provider defaults
      --extra-boot-kernel-args string            add extra kernel args to the initial boot from vmlinuz and initramfs
      --extra-disks int                          number of extra disks to create for each worker VM
      --extra-disks-drivers strings              driver for each extra disk (virtio, ide, ahci, scsi, nvme, megaraid)
      --extra-disks-serials strings              serials for each extra disk
      --extra-disks-size int                     default limit on disk size in MB (each VM) (default 5120)
      --extra-disks-tags strings                 tags for each extra disk (only used by virtiofs)
      --extra-uefi-search-paths strings          additional search paths for UEFI firmware (only applies when UEFI is enabled)
  -h, --help                                     help for dev
      --image-cache-path string                  path to image cache
      --image-cache-port uint16                  port on which to serve image cache (default 5000)
      --image-cache-tls-cert-file string         path to image cache TLS cert
      --image-cache-tls-key-file string          path to image cache TLS key
      --init-node-as-endpoint                    use init node as endpoint instead of any load balancer endpoint
      --initrd-path string                       initramfs image to use (default "_out/initramfs-${ARCH}.xz")
      --install-image string                     the installer image to use (default "ghcr.io/siderolabs/installer:v1.13.0-alpha.1-240-g3ac26b8f9")
      --ipv4                                     enable IPv4 network in the cluster (default true)
      --ipv6                                     enable IPv6 network in the cluster
      --ipxe-boot-script string                  iPXE boot script (URL) to use
      --iso-path string                          the ISO path to use for the initial boot
      --memory string(mb,gb)                     the limit on memory usage for each control plane/VM (default 2.0GiB)
      --memory-workers string(mb,gb)             the limit on memory usage for each worker/VM (default 2.0GiB)
      --mtu int                                  MTU of the cluster network (default 1500)
      --nameservers strings                      list of nameservers to use (default [8.8.8.8,1.1.1.1,2001:4860:4860::8888,2606:4700:4700::1111])
      --omni-api-endpoint string                 the Omni API endpoint (must include a scheme, a hostname and a join token, e.g. 'https://siderolink.omni.example?jointoken=foobar')
      --registry-insecure-skip-verify strings    list of registry hostnames to skip TLS verification for
      --registry-mirror strings                  list of registry mirrors to use in format: <registry host>=<mirror URL>
      --skip-injecting-config                    skip injecting config from embedded metadata server, write config files to current directory
      --skip-injecting-extra-cmdline             skip injecting extra kernel cmdline parameters via EFI vars through bootloader
      --talos-version string                     the desired Talos version to generate config for (default "v1.13.0-alpha.1-240-g3ac26b8f9")
      --talosconfig string                       The location to save the generated client configuration file to. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --uki-path string                          the UKI image path to use for the initial boot
      --usb-path string                          the USB stick image path to use for the initial boot
      --use-vip                                  use a virtual IP for the controlplane endpoint instead of the loadbalancer
      --user-volumes strings                     list of user volumes to create for each VM in format: <name1>:<size1>:<name2>:<size2>
      --vmlinuz-path string                      the compressed kernel image to use (default "_out/vmlinuz-${ARCH}")
      --wait                                     wait for the cluster to be ready before returning (default true)
      --wait-timeout duration                    timeout to wait for the cluster to be ready (default 20m0s)
      --wireguard-cidr string                    CIDR of the wireguard network
      --with-apply-config                        enable apply config when the VM is starting in maintenance mode
      --with-bootloader                          enable bootloader to load kernel and initramfs from disk image after install (default true)
      --with-cluster-discovery                   enable cluster discovery (default true)
      --with-debug                               enable debug in Talos config to send service logs to the console
      --with-firewall string                     inject firewall rules into the cluster, value is default policy - accept/block
      --with-init-node                           create the cluster with an init node
      --with-iommu                               enable IOMMU support, this also add a new PCI root port and an interface attached to it
      --with-json-logs                           enable JSON logs receiver and configure Talos to send logs there
      --with-siderolink true                     enables the use of siderolink agent as configuration apply mechanism. true or `wireguard` enables the agent, `tunnel` enables the agent with grpc tunneling (default none)
      --with-tpm1_2                              enable TPM 1.2 emulation support using swtpm
      --with-tpm2                                enable TPM 2.0 emulation support using swtpm
      --with-uefi                                enable UEFI on x86_64 architecture (default true)
      --with-uuid-hostnames                      use machine UUIDs as default hostnames
      --workers int                              the number of workers to create (default 1)
```

### Options inherited from parent commands

```
      --name string    the name of the cluster (default "chubo-default")
      --state string   directory path to store cluster state (default "/Users/francesco/.chubo/clusters")
```

### SEE ALSO

* [chuboctl cluster create](#chuboctl-cluster-create)	 - Create a local Talos cluster.

## chuboctl cluster create docker

Create a local Docker based cluster

```
chuboctl cluster create docker [flags]
```

### Options

```
      --config-patch stringArray                 patch generated machineconfigs (applied to all node types), use @file to read a patch from file
      --config-patch-controlplanes stringArray   patch generated machineconfigs (applied to 'controlplane' type)
      --config-patch-workers stringArray         patch generated machineconfigs (applied to 'worker' type)
      --cpus-controlplanes string                the share of CPUs as fraction for each control plane/VM (default "2.0")
      --cpus-workers string                      the share of CPUs as fraction for each worker/VM (default "2.0")
  -p, --exposed-ports string                     comma-separated list of ports/protocols to expose on init node. Ex -p <hostPort>:<containerPort>/<protocol (tcp or udp)>
  -h, --help                                     help for docker
      --host-ip string                           Host IP to forward exposed ports to (default "0.0.0.0")
      --image string                             the talos image to run (default "ghcr.io/siderolabs/talos:v1.13.0-alpha.1-240-g3ac26b8f9")
      --memory-controlplanes string(mb,gb)       the limit on memory usage for each control plane/VM (default 2.0GiB)
      --memory-workers string(mb,gb)             the limit on memory usage for each worker/VM (default 2.0GiB)
      --mount mount                              attach a mount to the container (docker --mount syntax)
      --subnet string                            Docker network subnet CIDR (default "10.5.0.0/24")
      --talosconfig-destination string           The location to save the generated client configuration file to. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --workers int                              the number of workers to create (default 1)
```

### Options inherited from parent commands

```
      --name string    the name of the cluster (default "chubo-default")
      --state string   directory path to store cluster state (default "/Users/francesco/.chubo/clusters")
```

### SEE ALSO

* [chuboctl cluster create](#chuboctl-cluster-create)	 - Create a local Talos cluster.

## chuboctl cluster create qemu

Create a local QEMU based Talos cluster.

### Synopsis

Create a local QEMU based Talos cluster.

Available presets:
  - iso: Configure Talos to boot from an ISO from the Image Factory.
  - iso-secureboot: Configure Talos for Secureboot via ISO. Only available on Linux hosts.
  - pxe: Configure Talos to boot via PXE from the Image Factory.
  - disk-image: Configure Talos to boot from a disk image from the Image Factory.
  - maintenance: Skip applying machine configuration and leave the machines in maintenance mode. The machine configuration files are written to the working directory.

Note: exactly one of 'iso', 'iso-secureboot', 'pxe' or 'disk-image' presets must be specified.


```
chuboctl cluster create qemu [flags]
```

### Options

```
      --cidr string                              CIDR of the cluster network (default "10.5.0.0/24")
      --config-patch stringArray                 patch generated machineconfigs (applied to all node types), use @file to read a patch from file
      --config-patch-controlplanes stringArray   patch generated machineconfigs (applied to 'controlplane' type)
      --config-patch-workers stringArray         patch generated machineconfigs (applied to 'worker' type)
      --controlplanes int                        the number of controlplanes to create (default 1)
      --cpus-controlplanes string                the share of CPUs as fraction for each control plane/VM (default "2.0")
      --cpus-workers string                      the share of CPUs as fraction for each worker/VM (default "2.0")
      --disks disks                              list of disks to create in format "<driver1>:<size1>" (disks after the first one are added only to worker machines) (default virtio:10GiB,virtio:6GiB)
  -h, --help                                     help for qemu
      --image-factory-url string                 image factory url (default "https://factory.talos.dev/")
      --memory-controlplanes string(mb,gb)       the limit on memory usage for each control plane/VM (default 2.0GiB)
      --memory-workers string(mb,gb)             the limit on memory usage for each worker/VM (default 2.0GiB)
      --omni-api-endpoint string                 the Omni API endpoint (must include a scheme, a hostname and a join token, e.g. 'https://siderolink.omni.example?jointoken=foobar')
      --presets strings                          list of presets to apply (default [iso])
      --schematic-id string                      image factory schematic id (defaults to an empty schematic)
      --talos-version string                     the desired talos version (default "v1.13.0-alpha.1-240-g3ac26b8f9")
      --talosconfig-destination string           The location to save the generated client configuration file to. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --workers int                              the number of workers to create (default 1)
```

### Options inherited from parent commands

```
      --name string    the name of the cluster (default "chubo-default")
      --state string   directory path to store cluster state (default "/Users/francesco/.chubo/clusters")
```

### SEE ALSO

* [chuboctl cluster create](#chuboctl-cluster-create)	 - Create a local Talos cluster.

## chuboctl cluster destroy

Destroys a local provisioned cluster

```
chuboctl cluster destroy [flags]
```

### Options

```
  -f, --force                                   force deletion of cluster directory if there were errors
  -h, --help                                    help for destroy
      --save-cluster-logs-archive-path string   save cluster logs archive to the specified file on destroy
      --save-support-archive-path string        save support archive to the specified file on destroy
```

### Options inherited from parent commands

```
      --name string    the name of the cluster (default "chubo-default")
      --state string   directory path to store cluster state (default "/Users/francesco/.chubo/clusters")
```

### SEE ALSO

* [chuboctl cluster](#chuboctl-cluster)	 - A collection of commands for managing local docker-based or QEMU-based clusters

## chuboctl cluster show

Shows info about a local provisioned cluster

```
chuboctl cluster show [flags]
```

### Options

```
  -h, --help                 help for show
      --provisioner string   cluster provisioner to use (default "docker")
```

### Options inherited from parent commands

```
      --name string    the name of the cluster (default "chubo-default")
      --state string   directory path to store cluster state (default "/Users/francesco/.chubo/clusters")
```

### SEE ALSO

* [chuboctl cluster](#chuboctl-cluster)	 - A collection of commands for managing local docker-based or QEMU-based clusters

## chuboctl cluster

A collection of commands for managing local docker-based or QEMU-based clusters

### Options

```
  -h, --help           help for cluster
      --name string    the name of the cluster (default "chubo-default")
      --state string   directory path to store cluster state (default "/Users/francesco/.chubo/clusters")
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes
* [chuboctl cluster create](#chuboctl-cluster-create)	 - Create a local Talos cluster.
* [chuboctl cluster destroy](#chuboctl-cluster-destroy)	 - Destroys a local provisioned cluster
* [chuboctl cluster show](#chuboctl-cluster-show)	 - Shows info about a local provisioned cluster

## chuboctl completion

Output shell completion code for the specified shell (bash, fish or zsh)

### Synopsis

Output shell completion code for the specified shell (bash, fish or zsh).
The shell code must be evaluated to provide interactive
completion of chuboctl commands.  This can be done by sourcing it from
the .bash_profile.

Note for zsh users: [1] zsh completions are only supported in versions of zsh >= 5.2

```
chuboctl completion SHELL [flags]
```

### Examples

```
# Installing bash completion on macOS using homebrew
## If running Bash 3.2 included with macOS
	brew install bash-completion
## or, if running Bash 4.1+
	brew install bash-completion@2
## If chuboctl is installed via homebrew, this should start working immediately.
## If you've installed via other means, you may need add the completion to your completion directory
	chuboctl completion bash > $(brew --prefix)/etc/bash_completion.d/chuboctl

# Installing bash completion on Linux
## If bash-completion is not installed on Linux, please install the 'bash-completion' package
## via your distribution's package manager.
## Load the chuboctl completion code for bash into the current shell
	source <(chuboctl completion bash)
## Write bash completion code to a file and source if from .bash_profile
	chuboctl completion bash > "${CHUBO_HOME:-$HOME/.chubo}/completion.bash.inc"
	printf '
		# chuboctl shell completion
		source "${CHUBO_HOME:-$HOME/.chubo}/completion.bash.inc"
		' >> $HOME/.bash_profile
	source $HOME/.bash_profile
# Load the chuboctl completion code for fish[1] into the current shell
	chuboctl completion fish | source
# Set the chuboctl completion code for fish[1] to autoload on startup
    chuboctl completion fish > ~/.config/fish/completions/chuboctl.fish
# Load the chuboctl completion code for zsh[1] into the current shell
	source <(chuboctl completion zsh)
# Set the chuboctl completion code for zsh[1] to autoload on startup
    chuboctl completion zsh > "${fpath[1]}/_chuboctl"
```

### Options

```
  -h, --help   help for completion
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl config add

Add a new context

```
chuboctl config add <context> [flags]
```

### Options

```
      --ca string    the path to the CA certificate
      --crt string   the path to the certificate
  -h, --help         help for add
      --key string   the path to the key
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)

## chuboctl config context

Set the current context

```
chuboctl config context <context> [flags]
```

### Options

```
  -h, --help   help for context
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)

## chuboctl config contexts

List defined contexts

```
chuboctl config contexts [flags]
```

### Options

```
  -h, --help   help for contexts
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)

## chuboctl config endpoint

Set the endpoint(s) for the current context

```
chuboctl config endpoint <endpoint>... [flags]
```

### Options

```
  -h, --help   help for endpoint
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)

## chuboctl config info

Show information about the current context

```
chuboctl config info [flags]
```

### Options

```
  -h, --help            help for info
  -o, --output string   output format (json|yaml|text). Default text. (default "text")
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)

## chuboctl config merge

Merge additional contexts from another client configuration file

### Synopsis

Contexts with the same name are renamed while merging configs.

```
chuboctl config merge <from> [flags]
```

### Options

```
  -h, --help   help for merge
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)

## chuboctl config new

Generate a new client configuration file

```
chuboctl config new [<path>] [flags]
```

### Options

```
      --crt-ttl duration   certificate TTL (default 8760h0m0s)
  -h, --help               help for new
      --roles strings      roles (default [os:admin])
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)

## chuboctl config node

Set the node(s) for the current context

```
chuboctl config node <endpoint>... [flags]
```

### Options

```
  -h, --help   help for node
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)

## chuboctl config remove

Remove contexts

```
chuboctl config remove <context> [flags]
```

### Options

```
      --dry-run     dry run
  -h, --help        help for remove
  -y, --noconfirm   do not ask for confirmation
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)

## chuboctl config

Manage the client configuration file (chuboconfig)

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for config
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes
* [chuboctl config add](#chuboctl-config-add)	 - Add a new context
* [chuboctl config context](#chuboctl-config-context)	 - Set the current context
* [chuboctl config contexts](#chuboctl-config-contexts)	 - List defined contexts
* [chuboctl config endpoint](#chuboctl-config-endpoint)	 - Set the endpoint(s) for the current context
* [chuboctl config info](#chuboctl-config-info)	 - Show information about the current context
* [chuboctl config merge](#chuboctl-config-merge)	 - Merge additional contexts from another client configuration file
* [chuboctl config new](#chuboctl-config-new)	 - Generate a new client configuration file
* [chuboctl config node](#chuboctl-config-node)	 - Set the node(s) for the current context
* [chuboctl config remove](#chuboctl-config-remove)	 - Remove contexts

## chuboctl consulconfig

Download the Consul client configuration bundle from the node

### Synopsis

Download the Consul client configuration bundle from the node.

By default the bundle is extracted to PWD/consulconfig/.
If [local-path] is a directory, bundle is extracted under [local-path]/consulconfig/.
If [local-path] does not exist, it is created and used as the extraction directory.
If [local-path] is "-", the raw .tar.gz bundle is written to stdout.

```
chuboctl consulconfig [local-path] [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -f, --force                      Force overwrite if the output file already exists
  -h, --help                       help for consulconfig
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl containers

List containers

```
chuboctl containers [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for containers
  -k, --kubernetes                 use the k8s.io containerd namespace
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl copy

Copy data out from the node

### Synopsis

Creates an .tar.gz archive at the node starting at <src-path> and
streams it back to the client.

If '-' is given for <local-path>, archive is written to stdout.
Otherwise archive is extracted to <local-path> which should be an empty directory or
talosctl creates a directory if <local-path> doesn't exist. Command doesn't preserve
ownership and access mode for the files in extract mode, while  streamed .tar archive
captures ownership and permission bits.

```
chuboctl copy <src-path> -|<local-path> [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for copy
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl dashboard

Cluster dashboard with node overview, logs and real-time metrics

### Synopsis

Provide a text-based UI to navigate node overview, logs and real-time metrics.

Keyboard shortcuts:

 - h, <Left> - switch one node to the left
 - l, <Right> - switch one node to the right
 - j, <Down> - scroll logs/process list down
 - k, <Up> - scroll logs/process list up
 - <C-d> - scroll logs/process list half page down
 - <C-u> - scroll logs/process list half page up
 - <C-f> - scroll logs/process list one page down
 - <C-b> - scroll logs/process list one page up


```
chuboctl dashboard [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for dashboard
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -d, --update-interval duration   interval between updates (default 3s)
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl debug

Run a debug container from an image archive or reference

```
chuboctl debug <image-tar-path|image ref> [args] [flags]
```

### Examples

```
  # Run a debug container from a local tar archive (image will be loaded into Talos from the archive)
    talosctl debug ./debug-tools.tar --args /bin/sh

  # Run a debug container from an image reference (Talos will pull the image if not present)
    talosctl debug docker.io/library/alpine:latest --args /bin/sh
```

### Options

```
      --args strings               arguments to pass to the container
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for debug
      --namespace system           namespace to use: system (CRI containerd) or `inmem` for in-memory containerd instance (default "inmem")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl dmesg

Retrieve kernel logs

```
chuboctl dmesg [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -f, --follow                     specify if the kernel log should be streamed
  -h, --help                       help for dmesg
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --tail                       specify if only new messages should be sent (makes sense only when combined with --follow)
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl edit

Edit machine configuration with the default editor.

### Synopsis

The edit command allows you to directly edit the machine configuration
of a Talos node using your preferred text editor.

It will open the editor defined by your CHUBO_EDITOR,
TALOS_EDITOR, or EDITOR environment variables, or fall back to 'vi' for Linux
or 'notepad' for Windows.

```
chuboctl edit machineconfig [flags]
```

### Options

```
      --chuboconfig string                          The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string                              Cluster to connect to if a proxy endpoint is used.
      --context string                              Context to be used in command
      --dry-run                                     do not apply the change after editing and print the change summary instead
  -e, --endpoints strings                           override default endpoints in client configuration
  -h, --help                                        help for edit
  -m, --mode auto, no-reboot, reboot, staged, try   apply config mode (default auto)
      --namespace string                            resource namespace (default is to use default namespace per resource)
  -n, --nodes strings                               target the specified nodes
      --siderov1-keys-dir string                    The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string                          Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --timeout duration                            the config will be rolled back after specified timeout (if try mode is selected) (default 1m0s)
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl events

Stream runtime events

```
chuboctl events [flags]
```

### Options

```
      --actor-id string            filter events by the specified actor ID (default is no filter)
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
      --duration duration          show events for the past duration interval (one second resolution, default is to show no history)
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for events
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --since string               show events after the specified event ID (default is to show no history)
      --tail int32                 show specified number of past events (use -1 to show full history, default is to show no history)
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl gen ca

Generates a self-signed X.509 certificate authority

```
chuboctl gen ca [flags]
```

### Options

```
  -h, --help                  help for ca
      --hours int             the hours from now on which the certificate validity period ends (default 87600)
      --organization string   X.509 distinguished name for the Organization
      --rsa                   generate in RSA format
```

### Options inherited from parent commands

```
  -f, --force   will overwrite existing files
```

### SEE ALSO

* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys

## chuboctl gen config

Generates a set of configuration files for Talos cluster

### Synopsis

The cluster endpoint is the URL for the Kubernetes API. If you decide to use
a control plane node, common in a single node control plane setup, use port 6443 as
this is the port that the API server binds to on every control plane node. For an HA
setup, usually involving a load balancer, use the IP and port of the load balancer.

```
chuboctl gen config <cluster name> <cluster endpoint> [flags]
```

### Options

```
      --additional-sans strings                  additional Subject-Alt-Names for the APIServer certificate
      --config-patch stringArray                 patch generated machineconfigs (applied to all node types), use @file to read a patch from file
      --config-patch-control-plane stringArray   patch generated machineconfigs (applied to 'init' and 'controlplane' types)
      --config-patch-worker stringArray          patch generated machineconfigs (applied to 'worker' type)
      --dns-domain string                        the dns domain to use for cluster (default "cluster.local")
  -h, --help                                     help for config
      --install-disk string                      the disk to install to (default "/dev/sda")
      --install-image string                     the image used to perform an installation (default "ghcr.io/siderolabs/installer:v1.13.0-alpha.1-240-g3ac26b8f9")
      --kubernetes-version string                desired kubernetes version to run (default "1.35.0")
  -o, --output string                            destination to output generated files. when multiple output types are specified, it must be a directory. for a single output type, it must either be a file path, or "-" for stdout
  -t, --output-types strings                     types of outputs to be generated. valid types are: ["controlplane" "worker" "talosconfig"] (default [controlplane,worker,talosconfig])
      --registry-mirror strings                  list of registry mirrors to use in format: <registry host>=<mirror URL>
      --talos-version string                     the desired Talos version to generate config for (backwards compatibility, e.g. v0.8)
      --version string                           the desired machine config version to generate (default "v1alpha1")
      --with-cluster-discovery                   enable cluster discovery feature (default true)
      --with-docs                                renders all machine configs adding the documentation for each field (default true)
      --with-examples                            renders all machine configs with the commented examples (default true)
      --with-secrets string                      use a secrets file generated using 'gen secrets'
```

### Options inherited from parent commands

```
  -f, --force   will overwrite existing files
```

### SEE ALSO

* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys

## chuboctl gen crt

Generates an X.509 Ed25519 certificate

```
chuboctl gen crt [flags]
```

### Options

```
      --ca string     path to the PEM encoded CERTIFICATE
      --csr string    path to the PEM encoded CERTIFICATE REQUEST
  -h, --help          help for crt
      --hours int     the hours from now on which the certificate validity period ends (default 24)
      --name string   the basename of the generated file
```

### Options inherited from parent commands

```
  -f, --force   will overwrite existing files
```

### SEE ALSO

* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys

## chuboctl gen csr

Generates a CSR using an Ed25519 private key

```
chuboctl gen csr [flags]
```

### Options

```
  -h, --help            help for csr
      --ip string       generate the certificate for this IP address
      --key string      path to the PEM encoded EC or RSA PRIVATE KEY
      --roles strings   roles (default [os:admin])
```

### Options inherited from parent commands

```
  -f, --force   will overwrite existing files
```

### SEE ALSO

* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys

## chuboctl gen key

Generates an Ed25519 private key

```
chuboctl gen key [flags]
```

### Options

```
  -h, --help          help for key
      --name string   the basename of the generated file
```

### Options inherited from parent commands

```
  -f, --force   will overwrite existing files
```

### SEE ALSO

* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys

## chuboctl gen keypair

Generates an X.509 Ed25519 key pair

```
chuboctl gen keypair [flags]
```

### Options

```
  -h, --help                  help for keypair
      --ip string             generate the certificate for this IP address
      --organization string   X.509 distinguished name for the Organization
```

### Options inherited from parent commands

```
  -f, --force   will overwrite existing files
```

### SEE ALSO

* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys

## chuboctl gen machineconfig

Generate a minimal (non-Kubernetes) machine config for Chubo

### Synopsis

Generates a single YAML document:

  apiVersion: chubo.dev/v1alpha1
  kind: MachineConfig

The output is suitable for `talosctl apply-config` in the `chubo` build variant.


```
chuboctl gen machineconfig [flags]
```

### Options

```
      --chubo-bootstrap-expect int       bootstrap_expect for openwonton/opengyoza (unset by default) (default -1)
      --chubo-join strings               peer addresses to join/retry-join for openwonton/opengyoza
      --chubo-role string                chubo role for openwonton/opengyoza (server|client) (default "server")
  -h, --help                             help for machineconfig
      --id string                        optional stable node id (metadata.id)
      --install-disk string              disk to install to (default "/dev/sda")
      --install-image string             installer image to install from (leave empty if you set it via boot args)
      --openbao-mode string              openbao mode when enabled (nomadJob) (default "nomadJob")
      --opengyoza-artifact-url string    override opengyoza artifact URL (http(s)://...)
      --openwonton-artifact-url string   override openwonton artifact URL (http(s)://...)
  -o, --output string                    output path, or "-" for stdout (default "-")
      --registry-mirror strings          registry mirrors in format: <registry host>=<mirror URL>
      --wipe                             wipe the install disk before installing (default true)
      --with-chubo                       enable modules.chubo with openwonton/opengyoza defaults
      --with-openbao                     enable modules.chubo.openbao (Nomad job controller)
      --with-secrets string              use a secrets file generated using 'gen secrets' (optional)
```

### Options inherited from parent commands

```
  -f, --force   will overwrite existing files
```

### SEE ALSO

* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys

## chuboctl gen secrets

Generates a secrets bundle file which can later be used to generate a config

```
chuboctl gen secrets [flags]
```

### Options

```
      --from-controlplane-config string     use the provided controlplane Talos machine configuration as input
  -p, --from-kubernetes-pki string          use a Kubernetes PKI directory (e.g. /etc/kubernetes/pki) as input
  -h, --help                                help for secrets
  -t, --kubernetes-bootstrap-token string   use the provided bootstrap token as input
  -o, --output-file string                  path of the output file, or "-" for stdout (default "secrets.yaml")
      --talos-version string                the desired Talos version to generate secrets bundle for (backwards compatibility, e.g. v0.8)
```

### Options inherited from parent commands

```
  -f, --force   will overwrite existing files
```

### SEE ALSO

* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys

## chuboctl gen secureboot database

Generates a UEFI database to enroll the signing certificate

```
chuboctl gen secureboot database [flags]
```

### Options

```
      --enrolled-certificate string     path to the certificate to enroll (default "_out/uki-signing-cert.pem")
  -h, --help                            help for database
      --include-well-known-uefi-certs   include well-known UEFI (Microsoft) certificates in the database
      --signing-certificate string      path to the certificate used to sign the database (default "_out/uki-signing-cert.pem")
      --signing-key string              path to the key used to sign the database (default "_out/uki-signing-key.pem")
```

### Options inherited from parent commands

```
  -f, --force           will overwrite existing files
  -o, --output string   path to the directory storing the generated files (default "_out")
```

### SEE ALSO

* [chuboctl gen secureboot](#chuboctl-gen-secureboot)	 - Generates secrets for the SecureBoot process

## chuboctl gen secureboot pcr

Generates a key which is used to sign TPM PCR values

```
chuboctl gen secureboot pcr [flags]
```

### Options

```
  -h, --help   help for pcr
```

### Options inherited from parent commands

```
  -f, --force           will overwrite existing files
  -o, --output string   path to the directory storing the generated files (default "_out")
```

### SEE ALSO

* [chuboctl gen secureboot](#chuboctl-gen-secureboot)	 - Generates secrets for the SecureBoot process

## chuboctl gen secureboot uki

Generates a certificate which is used to sign boot assets (UKI)

```
chuboctl gen secureboot uki [flags]
```

### Options

```
      --common-name string   common name for the certificate (default "Test UKI Signing Key")
  -h, --help                 help for uki
```

### Options inherited from parent commands

```
  -f, --force           will overwrite existing files
  -o, --output string   path to the directory storing the generated files (default "_out")
```

### SEE ALSO

* [chuboctl gen secureboot](#chuboctl-gen-secureboot)	 - Generates secrets for the SecureBoot process

## chuboctl gen secureboot

Generates secrets for the SecureBoot process

### Options

```
  -h, --help            help for secureboot
  -o, --output string   path to the directory storing the generated files (default "_out")
```

### Options inherited from parent commands

```
  -f, --force   will overwrite existing files
```

### SEE ALSO

* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys
* [chuboctl gen secureboot database](#chuboctl-gen-secureboot-database)	 - Generates a UEFI database to enroll the signing certificate
* [chuboctl gen secureboot pcr](#chuboctl-gen-secureboot-pcr)	 - Generates a key which is used to sign TPM PCR values
* [chuboctl gen secureboot uki](#chuboctl-gen-secureboot-uki)	 - Generates a certificate which is used to sign boot assets (UKI)

## chuboctl gen

Generate CAs, certificates, and private keys

### Options

```
  -f, --force   will overwrite existing files
  -h, --help    help for gen
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes
* [chuboctl gen ca](#chuboctl-gen-ca)	 - Generates a self-signed X.509 certificate authority
* [chuboctl gen config](#chuboctl-gen-config)	 - Generates a set of configuration files for Talos cluster
* [chuboctl gen crt](#chuboctl-gen-crt)	 - Generates an X.509 Ed25519 certificate
* [chuboctl gen csr](#chuboctl-gen-csr)	 - Generates a CSR using an Ed25519 private key
* [chuboctl gen key](#chuboctl-gen-key)	 - Generates an Ed25519 private key
* [chuboctl gen keypair](#chuboctl-gen-keypair)	 - Generates an X.509 Ed25519 key pair
* [chuboctl gen machineconfig](#chuboctl-gen-machineconfig)	 - Generate a minimal (non-Kubernetes) machine config for Chubo
* [chuboctl gen secrets](#chuboctl-gen-secrets)	 - Generates a secrets bundle file which can later be used to generate a config
* [chuboctl gen secureboot](#chuboctl-gen-secureboot)	 - Generates secrets for the SecureBoot process

## chuboctl get

Get a specific resource or list of resources (use 'talosctl get rd' to see all available resource types).

### Synopsis

Similar to 'kubectl get', 'talosctl get' returns a set of resources from the OS.
To get a list of all available resource definitions, issue 'talosctl get rd'

```
chuboctl get <type> [<id>] [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for get
  -i, --insecure                   get resources using the insecure (encrypted with no auth) maintenance service
      --namespace string           resource namespace (default is to use default namespace per resource)
  -n, --nodes strings              target the specified nodes
  -o, --output string              output mode (json, table, yaml, jsonpath) (default "table")
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -w, --watch                      watch resource changes
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl image cache-cert-gen

Generate TLS certificates and CA patch required for securing image cache to Talos communication

### Synopsis

Generate TLS certificates and CA patch required for securing image cache to Talos communication

```
chuboctl image cache-cert-gen [flags]
```

### Options

```
      --advertised-address ipSlice   The addresses to advertise. (default [])
      --advertised-name strings      The DNS names to advertise.
  -h, --help                         help for cache-cert-gen
      --tls-ca-file string           TLS certificate authority file (default "ca.crt")
      --tls-cert-file string         TLS certificate file to use for serving (default "tls.crt")
      --tls-key-file string          TLS key file to use for serving (default "tls.key")
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
      --namespace system           namespace to use: system (etcd and kubelet images) or `cri` for all Kubernetes workloads, `inmem` for in-memory containerd instance (default "cri")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl image](#chuboctl-image)	 - Manage container images

## chuboctl image cache-create

Create a cache of images in OCI format into a directory

### Synopsis

Create a cache of images in OCI format into a directory

```
chuboctl image cache-create [flags]
```

### Examples

```
talosctl images cache-create --images=ghcr.io/siderolabs/kubelet:v1.35.0 --image-cache-path=/tmp/talos-image-cache

Alternatively, stdin can be piped to the command:
talosctl images default | talosctl images cache-create --image-cache-path=/tmp/talos-image-cache --images=-

```

### Options

```
      --force                           force overwrite of existing image cache
  -h, --help                            help for cache-create
      --image-cache-path string         directory to save the image cache in OCI format
      --image-layer-cache-path string   directory to save the image layer cache
      --images strings                  images to cache
      --insecure                        allow insecure registries
      --layout string                   Specifies the cache layout format: "oci" for an OCI image layout directory, or "flat" for a registry-like flat file structure (default "oci")
      --platform strings                platform to use for the cache (default [linux/amd64])
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
      --namespace system           namespace to use: system (etcd and kubelet images) or `cri` for all Kubernetes workloads, `inmem` for in-memory containerd instance (default "cri")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl image](#chuboctl-image)	 - Manage container images

## chuboctl image cache-serve

Serve an OCI image cache directory over HTTP(S) as a container registry

### Synopsis

Serve an OCI image cache directory over HTTP(S) as a container registry

```
chuboctl image cache-serve [flags]
```

### Options

```
      --address string            address to serve the registry on (default "127.0.0.1:3172")
  -h, --help                      help for cache-serve
      --image-cache-path string   directory to save the image cache in flat format
      --mirror strings            list of registry mirrors to add to the Talos config patch (default [docker.io,ghcr.io,registry.k8s.io])
      --tls-cert-file string      TLS certificate file to use for serving
      --tls-key-file string       TLS key file to use for serving
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
      --namespace system           namespace to use: system (etcd and kubelet images) or `cri` for all Kubernetes workloads, `inmem` for in-memory containerd instance (default "cri")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl image](#chuboctl-image)	 - Manage container images

## chuboctl image k8s-bundle

List the default Kubernetes images used by Talos

```
chuboctl image k8s-bundle [flags]
```

### Options

```
      --coredns-version semver   CoreDNS semantic version (default v1.13.2)
      --etcd-version semver      ETCD semantic version (default v3.6.7)
      --flannel-version semver   Flannel CNI semantic version (default v0.27.4)
  -h, --help                     help for k8s-bundle
      --k8s-version semver       Kubernetes semantic version (default v1.35.0)
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
      --namespace system           namespace to use: system (etcd and kubelet images) or `cri` for all Kubernetes workloads, `inmem` for in-memory containerd instance (default "cri")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl image](#chuboctl-image)	 - Manage container images

## chuboctl image list

List images in the machine's container runtime

```
chuboctl image list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
      --namespace system           namespace to use: system (etcd and kubelet images) or `cri` for all Kubernetes workloads, `inmem` for in-memory containerd instance (default "cri")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl image](#chuboctl-image)	 - Manage container images

## chuboctl image pull

Pull an image into the machine's container runtime

```
chuboctl image pull <image> [flags]
```

### Options

```
  -h, --help   help for pull
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
      --namespace system           namespace to use: system (etcd and kubelet images) or `cri` for all Kubernetes workloads, `inmem` for in-memory containerd instance (default "cri")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl image](#chuboctl-image)	 - Manage container images

## chuboctl image remove

Remove an image from the machine's container runtime

```
chuboctl image remove <image> [flags]
```

### Options

```
  -h, --help   help for remove
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
      --namespace system           namespace to use: system (etcd and kubelet images) or `cri` for all Kubernetes workloads, `inmem` for in-memory containerd instance (default "cri")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl image](#chuboctl-image)	 - Manage container images

## chuboctl image talos-bundle

List the default system images and extensions used for Talos

```
chuboctl image talos-bundle [talos-version] [flags]
```

### Options

```
      --extensions   Include images that belong to Talos extensions (default true)
  -h, --help         help for talos-bundle
      --overlays     Include images that belong to Talos overlays (default true)
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
      --namespace system           namespace to use: system (etcd and kubelet images) or `cri` for all Kubernetes workloads, `inmem` for in-memory containerd instance (default "cri")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl image](#chuboctl-image)	 - Manage container images

## chuboctl image

Manage container images

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for image
      --namespace system           namespace to use: system (etcd and kubelet images) or `cri` for all Kubernetes workloads, `inmem` for in-memory containerd instance (default "cri")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes
* [chuboctl image cache-cert-gen](#chuboctl-image-cache-cert-gen)	 - Generate TLS certificates and CA patch required for securing image cache to Talos communication
* [chuboctl image cache-create](#chuboctl-image-cache-create)	 - Create a cache of images in OCI format into a directory
* [chuboctl image cache-serve](#chuboctl-image-cache-serve)	 - Serve an OCI image cache directory over HTTP(S) as a container registry
* [chuboctl image k8s-bundle](#chuboctl-image-k8s-bundle)	 - List the default Kubernetes images used by Talos
* [chuboctl image list](#chuboctl-image-list)	 - List images in the machine's container runtime
* [chuboctl image pull](#chuboctl-image-pull)	 - Pull an image into the machine's container runtime
* [chuboctl image remove](#chuboctl-image-remove)	 - Remove an image from the machine's container runtime
* [chuboctl image talos-bundle](#chuboctl-image-talos-bundle)	 - List the default system images and extensions used for Talos

## chuboctl inspect dependencies

Inspect controller-resource dependencies as graphviz graph.

### Synopsis

Inspect controller-resource dependencies as graphviz graph.

Pipe the output of the command through the "dot" program (part of graphviz package)
to render the graph:

    talosctl inspect dependencies | dot -Tpng > graph.png


```
chuboctl inspect dependencies [flags]
```

### Options

```
  -h, --help             help for dependencies
      --with-resources   display live resource information with dependencies
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl inspect](#chuboctl-inspect)	 - Inspect internals of the node OS

## chuboctl inspect

Inspect internals of the node OS

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for inspect
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes
* [chuboctl inspect dependencies](#chuboctl-inspect-dependencies)	 - Inspect controller-resource dependencies as graphviz graph.

## chuboctl list

Retrieve a directory listing

```
chuboctl list [path] [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -d, --depth int32                maximum recursion depth (default 1)
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for list
  -H, --humanize                   humanize size and time in the output
  -l, --long                       display additional file details
  -n, --nodes strings              target the specified nodes
  -r, --recurse                    recurse into subdirectories
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -t, --type strings               filter by specified types:
                                   f	regular file
                                   d	directory
                                   l, L	symbolic link
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl logs

Retrieve logs for a service

```
chuboctl logs <service name> [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -f, --follow                     specify if the logs should be streamed
  -h, --help                       help for logs
  -k, --kubernetes                 use the k8s.io containerd namespace
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --tail int32                 lines of log file to display (default is to show from the beginning) (default -1)
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl machineconfig gen

Generates a set of configuration files for Talos cluster

### Synopsis

The cluster endpoint is the URL for the Kubernetes API. If you decide to use
a control plane node, common in a single node control plane setup, use port 6443 as
this is the port that the API server binds to on every control plane node. For an HA
setup, usually involving a load balancer, use the IP and port of the load balancer.

```
chuboctl machineconfig gen <cluster name> <cluster endpoint> [flags]
```

### Options

```
  -h, --help   help for gen
```

### SEE ALSO

* [chuboctl machineconfig](#chuboctl-machineconfig)	 - Machine config related commands

## chuboctl machineconfig patch

Patch a machine config

```
chuboctl machineconfig patch <machineconfig-file> [flags]
```

### Options

```
  -h, --help                help for patch
  -o, --output string       output destination. if not specified, output will be printed to stdout
  -p, --patch stringArray   patch generated machineconfigs (applied to all node types), use @file to read a patch from file
```

### SEE ALSO

* [chuboctl machineconfig](#chuboctl-machineconfig)	 - Machine config related commands

## chuboctl machineconfig

Machine config related commands

### Options

```
  -h, --help   help for machineconfig
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes
* [chuboctl machineconfig gen](#chuboctl-machineconfig-gen)	 - Generates a set of configuration files for Talos cluster
* [chuboctl machineconfig patch](#chuboctl-machineconfig-patch)	 - Patch a machine config

## chuboctl memory

Show memory usage

```
chuboctl memory [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for memory
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -v, --verbose                    display extended memory statistics
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl meta delete

Delete a key from the META partition.

```
chuboctl meta delete key [flags]
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -i, --insecure                   write|delete meta using the insecure (encrypted with no auth) maintenance service
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl meta](#chuboctl-meta)	 - Write and delete keys in the META partition

## chuboctl meta write

Write a key-value pair to the META partition.

```
chuboctl meta write key value [flags]
```

### Options

```
  -h, --help   help for write
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -i, --insecure                   write|delete meta using the insecure (encrypted with no auth) maintenance service
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl meta](#chuboctl-meta)	 - Write and delete keys in the META partition

## chuboctl meta

Write and delete keys in the META partition

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for meta
  -i, --insecure                   write|delete meta using the insecure (encrypted with no auth) maintenance service
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes
* [chuboctl meta delete](#chuboctl-meta-delete)	 - Delete a key from the META partition.
* [chuboctl meta write](#chuboctl-meta-write)	 - Write a key-value pair to the META partition.

## chuboctl mounts

List mounts

```
chuboctl mounts [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for mounts
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl netstat

Show network connections and sockets

### Synopsis

Show network connections and sockets.

You can pass an optional argument to view a specific pod's connections.
To do this, format the argument as "namespace/pod".
Note that only pods with a pod network namespace are allowed.
If you don't pass an argument, the command will show host connections.

```
chuboctl netstat [flags]
```

### Options

```
  -a, --all                        display all sockets states (default: connected)
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -x, --extend                     show detailed socket information
  -h, --help                       help for netstat
  -4, --ipv4                       display only ipv4 sockets
  -6, --ipv6                       display only ipv6 sockets
  -l, --listening                  display listening server sockets
  -n, --nodes strings              target the specified nodes
  -k, --pods                       show sockets used by Kubernetes pods
  -p, --programs                   show process using socket
  -w, --raw                        display only RAW sockets
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -t, --tcp                        display only TCP sockets
  -o, --timers                     display timers
  -u, --udp                        display only UDP sockets
  -U, --udplite                    display only UDPLite sockets
  -v, --verbose                    display sockets of all supported transport protocols
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl nomadconfig

Download the Nomad client configuration bundle from the node

### Synopsis

Download the Nomad client configuration bundle from the node.

By default the bundle is extracted to PWD/nomadconfig/.
If [local-path] is a directory, bundle is extracted under [local-path]/nomadconfig/.
If [local-path] does not exist, it is created and used as the extraction directory.
If [local-path] is "-", the raw .tar.gz bundle is written to stdout.

```
chuboctl nomadconfig [local-path] [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -f, --force                      Force overwrite if the output file already exists
  -h, --help                       help for nomadconfig
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl openbaoconfig

Download the OpenBao client configuration bundle from the node

### Synopsis

Download the OpenBao client configuration bundle from the node.

By default the bundle is extracted to PWD/openbaoconfig/.
If [local-path] is a directory, bundle is extracted under [local-path]/openbaoconfig/.
If [local-path] does not exist, it is created and used as the extraction directory.
If [local-path] is "-", the raw .tar.gz bundle is written to stdout.

```
chuboctl openbaoconfig [local-path] [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -f, --force                      Force overwrite if the output file already exists
  -h, --help                       help for openbaoconfig
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl patch

Patch machine configuration of a node with a local patch.

```
chuboctl patch machineconfig [flags]
```

### Options

```
      --chuboconfig string                          The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string                              Cluster to connect to if a proxy endpoint is used.
      --context string                              Context to be used in command
      --dry-run                                     print the change summary and patch preview without applying the changes
  -e, --endpoints strings                           override default endpoints in client configuration
  -h, --help                                        help for patch
  -m, --mode auto, no-reboot, reboot, staged, try   apply config mode (default auto)
      --namespace string                            resource namespace (default is to use default namespace per resource)
  -n, --nodes strings                               target the specified nodes
  -p, --patch stringArray                           the patch to be applied to the resource file, use @file to read a patch from file.
      --patch-file string                           a file containing a patch to be applied to the resource.
      --siderov1-keys-dir string                    The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string                          Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --timeout duration                            the config will be rolled back after specified timeout (if try mode is selected) (default 1m0s)
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl pcap

Capture the network packets from the node.

### Synopsis

The command launches packet capture on the node and streams back the packets as raw pcap file.

```
chuboctl pcap [flags]
```

### Examples

```
Default behavior is to decode the packets with internal decoder to stdout:

    talosctl pcap -i eth0

Raw pcap file can be saved with `--output` flag:

    talosctl pcap -i eth0 --output eth0.pcap

Output can be piped to tcpdump:

    talosctl pcap -i eth0 -o - | tcpdump -vvv -r -

BPF filter can be applied, but it has to compiled to BPF instructions first using tcpdump.
Correct link type should be specified for the tcpdump: EN10MB for Ethernet links and RAW
for e.g. Wireguard tunnels:

    talosctl pcap -i eth0 --bpf-filter "$(tcpdump -dd -y EN10MB 'tcp and dst port 80')"

	    talosctl pcap -i siderolink --bpf-filter "$(tcpdump -dd -y RAW 'port 50000')"

	As packet capture is transmitted over the network, it is recommended to filter out the Talos API traffic,
	e.g. by excluding packets with the port 50000.
	   
```

### Options

```
      --bpf-filter string          bpf filter to apply, tcpdump -dd format
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
      --duration duration          duration of the capture
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for pcap
  -i, --interface string           interface name to capture packets on (default "eth0")
  -n, --nodes strings              target the specified nodes
  -o, --output string              if not set, decode packets to stdout; if set write raw pcap data to a file, use '-' for stdout
      --promiscuous                put interface into promiscuous mode
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl processes

List running processes

```
chuboctl processes [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for processes
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
  -s, --sort string                Column to sort output by. [rss|cpu] (default "rss")
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -w, --watch                      Stream running processes
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl read

Read a file on the machine

```
chuboctl read <path> [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for read
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl reboot

Reboot a node

```
chuboctl reboot [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
      --debug                      debug operation from kernel logs. --wait is set to true when this flag is set
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for reboot
  -m, --mode string                select the reboot mode: "default", "powercycle" (skips kexec), "force" (skips graceful teardown) (default "default")
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --timeout duration           time to wait for the operation is complete if --debug or --wait is set (default 30m0s)
      --wait                       wait for the operation to complete, tracking its progress. always set to true when --debug is set (default true)
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl reset

Reset a node

```
chuboctl reset [flags]
```

### Options

```
      --chuboconfig string                       The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string                           Cluster to connect to if a proxy endpoint is used.
      --context string                           Context to be used in command
      --debug                                    debug operation from kernel logs. --wait is set to true when this flag is set
  -e, --endpoints strings                        override default endpoints in client configuration
      --graceful                                 if true, attempt to cordon/drain node and leave etcd (if applicable) (default true)
  -h, --help                                     help for reset
      --insecure                                 reset using the insecure (encrypted with no auth) maintenance service
  -n, --nodes strings                            target the specified nodes
      --reboot                                   if true, reboot the node after resetting instead of shutting down
      --siderov1-keys-dir string                 The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --system-labels-to-wipe strings            if set, just wipe selected system disk partitions by label but keep other partitions intact
      --talosconfig string                       Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --timeout duration                         time to wait for the operation is complete if --debug or --wait is set (default 30m0s)
      --user-disks-to-wipe strings               if set, wipes defined devices in the list
      --wait                                     wait for the operation to complete, tracking its progress. always set to true when --debug is set (default true)
      --wipe-mode all, system-disk, user-disks   disk reset mode (default all)
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl restart

Restart a process

```
chuboctl restart <id> [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for restart
  -k, --kubernetes                 use the k8s.io containerd namespace
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl rollback

Rollback a node to the previous installation

```
chuboctl rollback [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for rollback
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl service

Retrieve the state of a service (or all services), control service state

### Synopsis

Service control command. If run without arguments, lists all the services and their state.
If service ID is specified, default action 'status' is executed which shows status of a single list service.
With actions 'start', 'stop', 'restart', service state is updated respectively.

```
chuboctl service [<id> [start|stop|restart|status]] [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for service
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl shutdown

Shutdown a node

```
chuboctl shutdown [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
      --debug                      debug operation from kernel logs. --wait is set to true when this flag is set
  -e, --endpoints strings          override default endpoints in client configuration
      --force                      if true, force a node to shutdown without a cordon/drain
  -h, --help                       help for shutdown
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --timeout duration           time to wait for the operation is complete if --debug or --wait is set (default 30m0s)
      --wait                       wait for the operation to complete, tracking its progress. always set to true when --debug is set (default true)
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl stats

Get container stats

```
chuboctl stats [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for stats
  -k, --kubernetes                 use the k8s.io containerd namespace
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl support

Dump debug information about the cluster

### Synopsis

Generated bundle contains the following debug information:

- For each node:

	- Kernel logs.
	- All Talos internal services logs.
	- All kube-system pods logs.
	- Talos COSI resources without secrets.
	- COSI runtime state graph.
	- Processes snapshot.
	- IO pressure snapshot.
	- Mounts list.
	- PCI devices info.
	- Talos version.

- For the cluster:

	- Kubernetes nodes and kube-system pods manifests.


```
chuboctl support [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for support
  -n, --nodes strings              target the specified nodes
  -w, --num-workers int            number of workers per node (default 1)
  -O, --output string              output file to write support archive to
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -v, --verbose                    verbose output
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl time

Gets current server time

```
chuboctl time [--check server] [flags]
```

### Options

```
      --check string               checks server time against specified ntp server
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for time
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl upgrade

Upgrade the node OS on the target node

```
chuboctl upgrade [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
      --debug                      debug operation from kernel logs. --wait is set to true when this flag is set
  -e, --endpoints strings          override default endpoints in client configuration
  -f, --force                      force the upgrade (skip checks on etcd health and members, might lead to data loss)
  -h, --help                       help for upgrade
  -i, --image string               the container image to use for performing the install (default "ghcr.io/siderolabs/installer:v1.13.0-alpha.1")
      --insecure                   upgrade using the insecure (encrypted with no auth) maintenance service
  -n, --nodes strings              target the specified nodes
  -m, --reboot-mode string         select the reboot mode during upgrade. Mode "powercycle" bypasses kexec. Valid values are: ["default" "powercycle"]. (default "default")
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
  -s, --stage                      stage the upgrade to perform it after a reboot
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --timeout duration           time to wait for the operation is complete if --debug or --wait is set (default 30m0s)
      --wait                       wait for the operation to complete, tracking its progress. always set to true when --debug is set (default true)
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl usage

Retrieve a disk usage

```
chuboctl usage [path1] [path2] ... [pathN] [flags]
```

### Options

```
  -a, --all                        write counts for all files, not just directories
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -d, --depth int32                maximum recursion depth
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for usage
  -H, --humanize                   humanize size and time in the output
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -t, --threshold int              threshold exclude entries smaller than SIZE if positive, or entries greater than SIZE if negative
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl validate

Validate config

```
chuboctl validate [flags]
```

### Options

```
  -c, --config string   the path of the config file
  -h, --help            help for validate
  -m, --mode string     the mode to validate the config for (valid values are metal, cloud, and container)
      --strict          treat validation warnings as errors
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl version

Prints the version

```
chuboctl version [flags]
```

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
      --client                     Print client version only
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for version
  -i, --insecure                   use Talos maintenance mode API
  -n, --nodes strings              target the specified nodes
      --short                      Print the short version
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes

## chuboctl wipe disk

Wipe a block device (disk or partition) which is not used as a volume

### Synopsis

Wipe a block device (disk or partition) which is not used as a volume.

Use device names as arguments, for example: vda or sda5.

```
chuboctl wipe disk <device names>... [flags]
```

### Options

```
      --drop-partition   drop partition after wipe (if applicable)
  -h, --help             help for disk
  -i, --insecure         use Talos maintenance mode API
      --method string    wipe method to use [FAST ZEROES] (default "FAST")
```

### Options inherited from parent commands

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl wipe](#chuboctl-wipe)	 - Wipe block device or volumes

## chuboctl wipe

Wipe block device or volumes

### Options

```
      --chuboconfig string         The path to the Chubo configuration file. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
  -c, --cluster string             Cluster to connect to if a proxy endpoint is used.
      --context string             Context to be used in command
  -e, --endpoints strings          override default endpoints in client configuration
  -h, --help                       help for wipe
  -n, --nodes strings              target the specified nodes
      --siderov1-keys-dir string   The path to the SideroV1 auth PGP keys directory. Defaults to 'SIDEROV1_KEYS_DIR' env variable if set, otherwise '$HOME/.chubo/keys' and legacy '$HOME/.talos/keys'. Only valid for Contexts that use SideroV1 auth.
      --talosconfig string         Legacy alias for --chuboconfig. Defaults to 'CHUBOCONFIG' (or legacy 'TALOSCONFIG') env variables if set, otherwise '$HOME/.chubo/config', then legacy '$HOME/.talos/config', then '/var/run/secrets/talos.dev/config'.
```

### SEE ALSO

* [chuboctl](#chuboctl)	 - A CLI for out-of-band management of Chubo OS nodes
* [chuboctl wipe disk](#chuboctl-wipe-disk)	 - Wipe a block device (disk or partition) which is not used as a volume

## chuboctl

A CLI for out-of-band management of Chubo OS nodes

### Options

```
  -h, --help   help for chuboctl
```

### SEE ALSO

* [chuboctl apply-config](#chuboctl-apply-config)	 - Apply a new configuration to a node
* [chuboctl cgroups](#chuboctl-cgroups)	 - Retrieve cgroups usage information
* [chuboctl cluster](#chuboctl-cluster)	 - A collection of commands for managing local docker-based or QEMU-based clusters
* [chuboctl completion](#chuboctl-completion)	 - Output shell completion code for the specified shell (bash, fish or zsh)
* [chuboctl config](#chuboctl-config)	 - Manage the client configuration file (chuboconfig)
* [chuboctl consulconfig](#chuboctl-consulconfig)	 - Download the Consul client configuration bundle from the node
* [chuboctl containers](#chuboctl-containers)	 - List containers
* [chuboctl copy](#chuboctl-copy)	 - Copy data out from the node
* [chuboctl dashboard](#chuboctl-dashboard)	 - Cluster dashboard with node overview, logs and real-time metrics
* [chuboctl debug](#chuboctl-debug)	 - Run a debug container from an image archive or reference
* [chuboctl dmesg](#chuboctl-dmesg)	 - Retrieve kernel logs
* [chuboctl edit](#chuboctl-edit)	 - Edit machine configuration with the default editor.
* [chuboctl events](#chuboctl-events)	 - Stream runtime events
* [chuboctl gen](#chuboctl-gen)	 - Generate CAs, certificates, and private keys
* [chuboctl get](#chuboctl-get)	 - Get a specific resource or list of resources (use 'talosctl get rd' to see all available resource types).
* [chuboctl image](#chuboctl-image)	 - Manage container images
* [chuboctl inspect](#chuboctl-inspect)	 - Inspect internals of the node OS
* [chuboctl list](#chuboctl-list)	 - Retrieve a directory listing
* [chuboctl logs](#chuboctl-logs)	 - Retrieve logs for a service
* [chuboctl machineconfig](#chuboctl-machineconfig)	 - Machine config related commands
* [chuboctl memory](#chuboctl-memory)	 - Show memory usage
* [chuboctl meta](#chuboctl-meta)	 - Write and delete keys in the META partition
* [chuboctl mounts](#chuboctl-mounts)	 - List mounts
* [chuboctl netstat](#chuboctl-netstat)	 - Show network connections and sockets
* [chuboctl nomadconfig](#chuboctl-nomadconfig)	 - Download the Nomad client configuration bundle from the node
* [chuboctl openbaoconfig](#chuboctl-openbaoconfig)	 - Download the OpenBao client configuration bundle from the node
* [chuboctl patch](#chuboctl-patch)	 - Patch machine configuration of a node with a local patch.
* [chuboctl pcap](#chuboctl-pcap)	 - Capture the network packets from the node.
* [chuboctl processes](#chuboctl-processes)	 - List running processes
* [chuboctl read](#chuboctl-read)	 - Read a file on the machine
* [chuboctl reboot](#chuboctl-reboot)	 - Reboot a node
* [chuboctl reset](#chuboctl-reset)	 - Reset a node
* [chuboctl restart](#chuboctl-restart)	 - Restart a process
* [chuboctl rollback](#chuboctl-rollback)	 - Rollback a node to the previous installation
* [chuboctl service](#chuboctl-service)	 - Retrieve the state of a service (or all services), control service state
* [chuboctl shutdown](#chuboctl-shutdown)	 - Shutdown a node
* [chuboctl stats](#chuboctl-stats)	 - Get container stats
* [chuboctl support](#chuboctl-support)	 - Dump debug information about the cluster
* [chuboctl time](#chuboctl-time)	 - Gets current server time
* [chuboctl upgrade](#chuboctl-upgrade)	 - Upgrade the node OS on the target node
* [chuboctl usage](#chuboctl-usage)	 - Retrieve a disk usage
* [chuboctl validate](#chuboctl-validate)	 - Validate config
* [chuboctl version](#chuboctl-version)	 - Prints the version
* [chuboctl wipe](#chuboctl-wipe)	 - Wipe block device or volumes

