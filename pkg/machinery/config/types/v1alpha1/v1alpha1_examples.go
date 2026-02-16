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

func machineControlplaneExample() *MachineControlPlaneConfig {
	return &MachineControlPlaneConfig{
		MachineControllerManager: &MachineControllerManagerConfig{
			MachineControllerManagerDisabled: pointer.To(false),
		},
		MachineScheduler: &MachineSchedulerConfig{
			MachineSchedulerDisabled: pointer.To(true),
		},
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
		CNI: &CNIConfig{
			CNIName: constants.FlannelCNI,
		},
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

func clusterAPIServerExample() *APIServerConfig {
	return &APIServerConfig{
		ContainerImage: clusterAPIServerImageExample(),
		ExtraArgsConfig: Args{
			"feature-gates":                    ArgValue{strValue: "ServerSideApply=true"},
			"http2-max-streams-per-connection": ArgValue{strValue: "32"},
		},
		CertSANs: []string{
			"1.2.3.4",
			"4.5.6.7",
		},
	}
}

func clusterAPIServerImageExample() string {
	return "registry.k8s.io/kube-apiserver:v1.32.0"
}

func clusterControllerManagerExample() *ControllerManagerConfig {
	return &ControllerManagerConfig{
		ContainerImage: clusterControllerManagerImageExample(),
		ExtraArgsConfig: Args{
			"feature-gates": ArgValue{strValue: "ServerSideApply=true"},
		},
	}
}

func clusterControllerManagerImageExample() string {
	return "registry.k8s.io/kube-controller-manager:v1.32.0"
}

func clusterProxyExample() *ProxyConfig {
	return &ProxyConfig{
		ContainerImage: clusterProxyImageExample(),
		ExtraArgsConfig: Args{
			"proxy-mode": ArgValue{strValue: "iptables"},
		},
		ModeConfig: "ipvs",
	}
}

func clusterProxyImageExample() string {
	return "registry.k8s.io/kube-proxy:v1.32.0"
}

func clusterSchedulerExample() *SchedulerConfig {
	return &SchedulerConfig{
		ContainerImage: clusterSchedulerImageExample(),
		ExtraArgsConfig: Args{
			"feature-gates": ArgValue{strValue: "AllBeta=true"},
		},
	}
}

func clusterSchedulerImageExample() string {
	return "registry.k8s.io/kube-scheduler:v1.32.0"
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

func clusterCoreDNSExample() *CoreDNS {
	return &CoreDNS{
		CoreDNSImage: (&CoreDNS{}).Image(),
	}
}

func clusterExternalCloudProviderConfigExample() *ExternalCloudProviderConfig {
	return &ExternalCloudProviderConfig{
		ExternalEnabled: pointer.To(true),
		ExternalManifests: []string{
			"https://raw.githubusercontent.com/kubernetes/cloud-provider-aws/v1.20.0-alpha.0/manifests/rbac.yaml",
			"https://raw.githubusercontent.com/kubernetes/cloud-provider-aws/v1.20.0-alpha.0/manifests/aws-cloud-controller-manager-daemonset.yaml",
		},
	}
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

func clusterCustomCNIExample() *CNIConfig {
	return &CNIConfig{
		CNIName: constants.CustomCNI,
		CNIUrls: []string{
			"https://docs.projectcalico.org/archive/v3.20/manifests/canal.yaml",
		},
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

func admissionControlConfigExample() []*AdmissionPluginConfig {
	return []*AdmissionPluginConfig{
		{
			PluginName: "PodSecurity",
			PluginConfiguration: Unstructured{
				Object: map[string]any{
					"apiVersion": "pod-security.admission.config.k8s.io/v1alpha1",
					"kind":       "PodSecurityConfiguration",
					"defaults": map[string]any{
						"enforce":         "baseline",
						"enforce-version": "latest",
						"audit":           "restricted",
						"audit-version":   "latest",
						"warn":            "restricted",
						"warn-version":    "latest",
					},
					"exemptions": map[string]any{
						"usernames":      []any{},
						"runtimeClasses": []any{},
						"namespaces":     []any{"kube-system"},
					},
				},
			},
		},
	}
}

func authorizationConfigExample() []*AuthorizationConfigAuthorizerConfig {
	return []*AuthorizationConfigAuthorizerConfig{
		{
			AuthorizerType: "Webhook",
			AuthorizerName: "webhook",
			AuthorizerWebhook: Unstructured{
				Object: map[string]any{
					"timeout":                    "3s",
					"subjectAccessReviewVersion": "v1",
					"matchConditionSubjectAccessReviewVersion": "v1",
					"failurePolicy": "Deny",
					"connectionInfo": map[string]any{
						"type": "InClusterConfig",
					},
					"matchConditions": []map[string]any{
						{
							"expression": "has(request.resourceAttributes)",
						},
						{
							"expression": "!(\\'system:serviceaccounts:kube-system\\' in request.groups)",
						},
					},
				},
			},
		},
		{
			AuthorizerType: "Webhook",
			AuthorizerName: "in-cluster-authorizer",
			AuthorizerWebhook: Unstructured{
				Object: map[string]any{
					"timeout":                    "3s",
					"subjectAccessReviewVersion": "v1",
					"matchConditionSubjectAccessReviewVersion": "v1",
					"failurePolicy": "NoOpinion",
					"connectionInfo": map[string]any{
						"type": "InClusterConfig",
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
