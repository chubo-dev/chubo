// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha1

import (
	"net/url"
	"strings"

	"github.com/siderolabs/crypto/x509"
	"github.com/siderolabs/go-pointer"
	"go.yaml.in/yaml/v4"

	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func mustParseURL(uri string) *url.URL {
	u, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}

	return u
}

// this is using custom type to avoid generating full example with all the nested structs.
func configExample() any {
	return struct {
		Version string `yaml:"version"`
		Machine *yaml.Node
		Cluster *yaml.Node
	}{
		Version: "v1alpha1",
		Machine: &yaml.Node{Kind: yaml.ScalarNode, LineComment: "..."},
		Cluster: &yaml.Node{Kind: yaml.ScalarNode, LineComment: "..."},
	}
}

func machineConfigExample() any {
	return struct {
		Type    string
		Install *InstallConfig
	}{
		Type:    machine.TypeControlPlane.String(),
		Install: machineInstallExample(),
	}
}

func pemEncodedCertificateExample() *x509.PEMEncodedCertificateAndKey {
	return &x509.PEMEncodedCertificateAndKey{
		Crt: []byte("--- EXAMPLE CERTIFICATE ---"),
		Key: []byte("--- EXAMPLE KEY ---"),
	}
}

func pemEncodedKeyExample() *x509.PEMEncodedKey {
	return &x509.PEMEncodedKey{
		Key: []byte("--- EXAMPLE KEY ---"),
	}
}

func machineInstallExample() *InstallConfig {
	return &InstallConfig{
		InstallDisk:              "/dev/sda",
		InstallImage:             "ghcr.io/siderolabs/installer:latest",
		InstallWipe:              pointer.To(false),
		InstallGrubUseUKICmdline: pointer.To(true),
	}
}

func machineInstallDiskSelectorExample() *InstallDiskSelector {
	return &InstallDiskSelector{
		Model: "WDC*",
		Size: &InstallDiskSizeMatcher{
			condition: ">= 1TB",
		},
	}
}

func machineInstallDiskSizeMatcherExamples0() *InstallDiskSizeMatcher {
	return &InstallDiskSizeMatcher{
		condition: "4GB",
	}
}

func machineInstallDiskSizeMatcherExamples1() *InstallDiskSizeMatcher {
	return &InstallDiskSizeMatcher{
		condition: "> 1TB",
	}
}

func machineInstallDiskSizeMatcherExamples2() *InstallDiskSizeMatcher {
	return &InstallDiskSizeMatcher{
		condition: "<= 2TB",
	}
}

func machineFilesExample() []*MachineFile {
	return []*MachineFile{
		{
			FileContent:     "...",
			FilePermissions: 0o666,
			FilePath:        "/tmp/file.txt",
			FileOp:          "append",
		},
	}
}

func machineSysctlsExample() map[string]string {
	return map[string]string{
		"kernel.domainname":                   "talos.dev",
		"net.ipv4.ip_forward":                 "0",
		"net/ipv6/conf/eth0.100/disable_ipv6": "1",
	}
}

func machineSysfsExample() map[string]string {
	return map[string]string{
		"devices.system.cpu.cpu0.cpufreq.scaling_governor": "performance",
	}
}

func machineFeaturesExample() *FeaturesConfig {
	return &FeaturesConfig{
		DiskQuotaSupport: pointer.To(true),
	}
}

func machineUdevExample() *UdevConfig {
	return &UdevConfig{
		UdevRules: []string{"SUBSYSTEM==\"drm\", KERNEL==\"renderD*\", GROUP=\"44\", MODE=\"0660\""},
	}
}

func clusterConfigExample() any {
	return struct {
		ControlPlane *ControlPlaneConfig   `yaml:"controlPlane"`
		ClusterName  string                `yaml:"clusterName"`
		Network      *ClusterNetworkConfig `yaml:"network"`
	}{
		ControlPlane: clusterControlPlaneExample(),
		ClusterName:  "talos.local",
		Network:      clusterNetworkExample(),
	}
}

func clusterControlPlaneExample() *ControlPlaneConfig {
	return &ControlPlaneConfig{
		Endpoint: &Endpoint{
			&url.URL{
				Host:   "1.2.3.4",
				Scheme: "https",
			},
		},
		LocalAPIServerPort: 443,
	}
}

func clusterNetworkExample() *ClusterNetworkConfig {
	return &ClusterNetworkConfig{
		DNSDomain:     "cluster.local",
		PodSubnet:     []string{"10.244.0.0/16"},
		ServiceSubnet: []string{"10.96.0.0/12"},
	}
}

func resourcesConfigRequestsExample() Unstructured {
	return Unstructured{
		Object: map[string]any{
			"cpu":    1,
			"memory": "1Gi",
		},
	}
}

func resourcesConfigLimitsExample() Unstructured {
	return Unstructured{
		Object: map[string]any{
			"cpu":    2,
			"memory": "2500Mi",
		},
	}
}

func clusterEtcdExample() *EtcdConfig {
	return &EtcdConfig{
		ContainerImage: clusterEtcdImageExample(),
		EtcdExtraArgs: Args{
			"election-timeout": ArgValue{strValue: "5000"},
		},
		RootCA: pemEncodedCertificateExample(),
	}
}

func clusterEtcdImageExample() string {
	return "registry.k8s.io/etcd:3.5.17-0"
}

func clusterEtcdAdvertisedSubnetsExample() []string {
	return []string{"10.0.0.0/8"}
}

func machineSeccompExample() []*MachineSeccompProfile {
	return []*MachineSeccompProfile{
		{
			MachineSeccompProfileName: "audit.json",
			MachineSeccompProfileValue: Unstructured{
				Object: map[string]any{
					"defaultAction": "SCMP_ACT_LOG",
				},
			},
		},
	}
}

func clusterEndpointExample1() *Endpoint {
	return &Endpoint{
		mustParseURL("https://1.2.3.4:6443"),
	}
}

func clusterEndpointExample2() *Endpoint {
	return &Endpoint{
		mustParseURL("https://cluster1.internal:6443"),
	}
}

func clusterInlineManifestsExample() ClusterInlineManifests {
	return ClusterInlineManifests{
		{
			InlineManifestName: "namespace-ci",
			InlineManifestContents: strings.TrimSpace(`
apiVersion: v1
kind: Namespace
metadata:
	name: ci
`),
		},
	}
}

func clusterDiscoveryExample() ClusterDiscoveryConfig {
	return ClusterDiscoveryConfig{
		DiscoveryEnabled: pointer.To(true),
		DiscoveryRegistries: DiscoveryRegistriesConfig{
			RegistryService: RegistryServiceConfig{
				RegistryEndpoint: constants.DefaultDiscoveryServiceEndpoint,
			},
		},
	}
}

func loggingEndpointExample1() *Endpoint {
	return &Endpoint{
		mustParseURL("udp://127.0.0.1:12345"),
	}
}

func loggingEndpointExample2() *Endpoint {
	return &Endpoint{
		mustParseURL("tcp://1.2.3.4:12345"),
	}
}

func machineLoggingExample() LoggingConfig {
	return LoggingConfig{
		LoggingDestinations: []LoggingDestination{
			{
				LoggingEndpoint: loggingEndpointExample2(),
				LoggingFormat:   constants.LoggingFormatJSONLines,
			},
		},
	}
}

func machineKernelExample() *KernelConfig {
	return &KernelConfig{
		KernelModules: []*KernelModuleConfig{
			{
				ModuleName: "btrfs",
			},
		},
	}
}

func machinePodsExample() []Unstructured {
	return []Unstructured{
		{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "pod",
				"metadata": map[string]any{
					"name": "nginx",
				},
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "nginx",
							"image": "nginx",
						},
					},
				},
			},
		},
	}
}

func kubernetesTalosAPIAccessConfigExample() *KubernetesTalosAPIAccessConfig {
	return &KubernetesTalosAPIAccessConfig{
		AccessEnabled: pointer.To(true),
		AccessAllowedRoles: []string{
			"os:reader",
		},
		AccessAllowedKubernetesNamespaces: []string{
			"kube-system",
		},
	}
}

func machineBaseRuntimeSpecOverridesExample() Unstructured {
	return Unstructured{
		Object: map[string]any{
			"process": map[string]any{
				"rlimits": []map[string]any{
					{
						"type": "RLIMIT_NOFILE",
						"hard": 1024,
						"soft": 1024,
					},
				},
			},
		},
	}
}
