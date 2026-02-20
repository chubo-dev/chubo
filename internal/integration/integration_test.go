// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build integration

// Package integration_test contains core runners for integration tests
package integration_test

import (
	"flag"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/chubo-dev/chubo/internal/integration/api"
	"github.com/chubo-dev/chubo/internal/integration/base"
	"github.com/chubo-dev/chubo/internal/integration/cli"
	provision_test "github.com/chubo-dev/chubo/internal/integration/provision"
	"github.com/chubo-dev/chubo/pkg/images"
	clientconfig "github.com/chubo-dev/chubo/pkg/machinery/client/config"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/version"
	"github.com/chubo-dev/chubo/pkg/provision"
	"github.com/chubo-dev/chubo/pkg/provision/providers"
)

// Accumulated list of all the suites to run.
var allSuites []suite.TestingSuite

// Flag values.
var (
	failFast         bool
	trustedBoot      bool
	selinuxEnforcing bool
	extensionsQEMU   bool
	extensionsNvidia bool
	verifyUKIBooted  bool
	airgapped        bool
	virtiofsd        bool
	race             bool

	chuboconfigPath   string
	endpoint          string
	k8sEndpoint       string
	expectedVersion   string
	expectedGoVersion string
	chuboctlPath      string
	kubectlPath       string
	helmPath          string
	kubeStrPath       string
	provisionerName   string
	clusterName       string
	stateDir          string
	chuboImage        string
	csiTestName       string
	csiTestTimeout    string
)

// TestIntegration ...
//
//nolint:gocyclo
func TestIntegration(t *testing.T) {
	if chuboconfigPath == "" {
		t.Error("--chubo.config is not provided")
	}

	var (
		cluster     provision.Cluster
		provisioner provision.Provisioner
		err         error
	)

	if provisionerName != "" {
		// use provisioned cluster state as discovery source
		ctx := t.Context()

		provisioner, err = providers.Factory(ctx, provisionerName)
		if err != nil {
			t.Error("error initializing provisioner", err)
		}

		defer provisioner.Close() //nolint:errcheck

		cluster, err = provisioner.Reflect(ctx, clusterName, stateDir)
		if err != nil {
			t.Error("error reflecting cluster via provisioner", err)
		}

		if k8sEndpoint == "" && provisionerName == base.ProvisionerDocker {
			k8sEndpoint = cluster.Info().ControlPlaneEndpoint
		}
	}

	provision_test.DefaultSettings.CurrentVersion = expectedVersion

	for _, s := range allSuites {
		if configuredSuite, ok := s.(base.ConfiguredSuite); ok {
			configuredSuite.SetConfig(base.ChuboSuite{
				Endpoint:         endpoint,
				K8sEndpoint:      k8sEndpoint,
				Cluster:          cluster,
				ChuboconfigPath:  chuboconfigPath,
				Version:          expectedVersion,
				GoVersion:        expectedGoVersion,
				ChuboctlPath:     chuboctlPath,
				KubectlPath:      kubectlPath,
				HelmPath:         helmPath,
				KubeStrPath:      kubeStrPath,
				ExtensionsQEMU:   extensionsQEMU,
				ExtensionsNvidia: extensionsNvidia,
				TrustedBoot:      trustedBoot,
				SelinuxEnforcing: selinuxEnforcing,
				VerifyUKIBooted:  verifyUKIBooted,
				ChuboImage:       chuboImage,
				CSITestName:      csiTestName,
				CSITestTimeout:   csiTestTimeout,
				Airgapped:        airgapped,
				Virtiofsd:        virtiofsd,
				Race:             race,
			})
		}

		var suiteName string
		if namedSuite, ok := s.(base.NamedSuite); ok {
			suiteName = namedSuite.SuiteName()
		}

		t.Run(suiteName, func(tt *testing.T) {
			suite.Run(tt, s) //nolint:scopelint
		})

		if failFast && t.Failed() {
			t.Log("fastfail mode enabled, aborting on first failure")

			break
		}
	}
}

func registerBoolFlag(bind *bool, primaryName, legacyName string, defaultValue bool, usage string) {
	flag.BoolVar(bind, primaryName, defaultValue, usage)
	flag.BoolVar(bind, legacyName, defaultValue, fmt.Sprintf("Legacy alias for --%s.", primaryName))
}

func registerStringFlag(bind *string, primaryName, legacyName, defaultValue, usage string) {
	flag.StringVar(bind, primaryName, defaultValue, usage)
	flag.StringVar(bind, legacyName, defaultValue, fmt.Sprintf("Legacy alias for --%s.", primaryName))
}

func registerIntFlag(bind *int, primaryName, legacyName string, defaultValue int, usage string) {
	flag.IntVar(bind, primaryName, defaultValue, usage)
	flag.IntVar(bind, legacyName, defaultValue, fmt.Sprintf("Legacy alias for --%s.", primaryName))
}

func registerInt64Flag(bind *int64, primaryName, legacyName string, defaultValue int64, usage string) {
	flag.Int64Var(bind, primaryName, defaultValue, usage)
	flag.Int64Var(bind, legacyName, defaultValue, fmt.Sprintf("Legacy alias for --%s.", primaryName))
}

func registerUint64Flag(bind *uint64, primaryName, legacyName string, defaultValue uint64, usage string) {
	flag.Uint64Var(bind, primaryName, defaultValue, usage)
	flag.Uint64Var(bind, legacyName, defaultValue, fmt.Sprintf("Legacy alias for --%s.", primaryName))
}

func registerValueFlag(bind flag.Value, primaryName, legacyName, usage string) {
	flag.Var(bind, primaryName, usage)
	flag.Var(bind, legacyName, fmt.Sprintf("Legacy alias for --%s.", primaryName))
}

func init() {
	defaultChuboConfigs, _ := clientconfig.GetDefaultPaths() //nolint:errcheck

	defaultStateDir, err := clientconfig.GetChuboDirectory()
	if err == nil {
		defaultStateDir = filepath.Join(defaultStateDir, "clusters")
	}

	registerBoolFlag(&failFast, "chubo.failfast", "talos.failfast", false, "fail the test run on the first failed test")
	registerBoolFlag(&trustedBoot, "chubo.trustedboot", "talos.trustedboot", false, "enable tests for trusted boot mode")
	registerBoolFlag(&selinuxEnforcing, "chubo.enforcing", "talos.enforcing", false, "enable tests for SELinux enforcing mode")
	registerBoolFlag(&extensionsQEMU, "chubo.extensions.qemu", "talos.extensions.qemu", false, "enable tests for qemu extensions")
	registerBoolFlag(&extensionsNvidia, "chubo.extensions.nvidia", "talos.extensions.nvidia", false, "enable tests for nvidia extensions")
	registerBoolFlag(&race, "chubo.race", "talos.race", false, "skip tests that are incompatible with race detector")
	registerBoolFlag(&verifyUKIBooted, "chubo.verifyukibooted", "talos.verifyukibooted", true, "enable tests for verifying that the node was booted using a UKI")

	registerStringFlag(
		&chuboconfigPath,
		"chubo.config",
		"talos.config",
		defaultChuboConfigs[0].Path,
		fmt.Sprintf("The path to the Chubo configuration file. Defaults to '%s' (legacy '%s') env variable if set, otherwise '%s', then legacy '%s', then '%s' in order.",
			constants.ChuboConfigEnvVar,
			constants.TalosConfigEnvVar,
			filepath.Join("$HOME", constants.ChuboDir, constants.ChuboconfigFilename),
			filepath.Join("$HOME", constants.TalosDir, constants.TalosconfigFilename),
			filepath.Join(constants.ServiceAccountMountPath, constants.ChuboconfigFilename),
		),
	)
	registerStringFlag(&endpoint, "chubo.endpoint", "talos.endpoint", "", "endpoint to use (overrides config)")
	registerStringFlag(&k8sEndpoint, "chubo.k8sendpoint", "talos.k8sendpoint", "", "Kubernetes endpoint to use (overrides kubeconfig)")
	registerStringFlag(&provisionerName, "chubo.provisioner", "talos.provisioner", "", "cluster provisioner to use, if not set cluster state is disabled")
	registerStringFlag(&stateDir, "chubo.state", "talos.state", defaultStateDir, "directory path to store cluster state")
	registerStringFlag(&clusterName, "chubo.name", "talos.name", "chubo-default", "the name of the cluster")
	registerStringFlag(&expectedVersion, "chubo.version", "talos.version", version.Tag, "expected Chubo OS version")
	registerStringFlag(&expectedGoVersion, "chubo.go.version", "talos.go.version", constants.GoVersion, "expected Go version")
	registerStringFlag(&chuboctlPath, "chubo.chuboctlpath", "talos.talosctlpath", "chuboctl", "The path to 'chuboctl' binary")
	registerStringFlag(&kubectlPath, "chubo.kubectlpath", "talos.kubectlpath", "kubectl", "The path to 'kubectl' binary")
	registerStringFlag(&helmPath, "chubo.helmpath", "talos.helmpath", "helm", "The path to 'helm' binary")
	registerStringFlag(&kubeStrPath, "chubo.kubestrpath", "talos.kubestrpath", "kubestr", "The path to 'kubestr' binary")
	registerStringFlag(&chuboImage, "chubo.image", "talos.image", images.DefaultTalosImageRepository, "The default Chubo OS container image")
	registerStringFlag(&csiTestName, "chubo.csi", "talos.csi", "", "CSI test to run")
	registerStringFlag(&csiTestTimeout, "chubo.csi.timeout", "talos.csi.timeout", "15m", "CSI test timeout")
	registerBoolFlag(&airgapped, "chubo.airgapped", "talos.airgapped", false, "marker to skip tests that should not be run on airgapped clusters")
	registerBoolFlag(&virtiofsd, "chubo.virtiofsd", "talos.virtiofsd", false, "marker to skip tests that should not be run without virtiofsd")

	registerStringFlag(&provision_test.DefaultSettings.CIDR, "chubo.provision.cidr", "talos.provision.cidr", provision_test.DefaultSettings.CIDR, "CIDR to use to provision clusters (provision tests only)")
	registerValueFlag(&provision_test.DefaultSettings.RegistryMirrors, "chubo.provision.registry-mirror", "talos.provision.registry-mirror", "registry mirrors to use (provision tests only)")
	registerIntFlag(&provision_test.DefaultSettings.MTU, "chubo.provision.mtu", "talos.provision.mtu", provision_test.DefaultSettings.MTU, "MTU to use for cluster network (provision tests only)")
	registerInt64Flag(&provision_test.DefaultSettings.CPUs, "chubo.provision.cpu", "talos.provision.cpu", provision_test.DefaultSettings.CPUs, "CPU count for each VM (provision tests only)")
	registerInt64Flag(&provision_test.DefaultSettings.MemMB, "chubo.provision.mem", "talos.provision.mem", provision_test.DefaultSettings.MemMB, "memory (in MiB) for each VM (provision tests only)")
	registerUint64Flag(&provision_test.DefaultSettings.DiskGB, "chubo.provision.disk", "talos.provision.disk", provision_test.DefaultSettings.DiskGB, "disk size (in GiB) for each VM (provision tests only)")
	registerIntFlag(&provision_test.DefaultSettings.ControlplaneNodes, "chubo.provision.controlplanes", "talos.provision.controlplanes", provision_test.DefaultSettings.ControlplaneNodes, "controlplane node count (provision tests only)")
	registerIntFlag(&provision_test.DefaultSettings.WorkerNodes, "chubo.provision.workers", "talos.provision.workers", provision_test.DefaultSettings.WorkerNodes, "worker node count (provision tests only)")
	registerStringFlag(&provision_test.DefaultSettings.TargetInstallImageRegistry, "chubo.provision.target-installer-registry", "talos.provision.target-installer-registry",
		provision_test.DefaultSettings.TargetInstallImageRegistry, "image registry for target installer image (provision tests only)")
	registerStringFlag(&provision_test.DefaultSettings.CustomCNIURL, "chubo.provision.custom-cni-url", "talos.provision.custom-cni-url", provision_test.DefaultSettings.CustomCNIURL, "custom CNI URL for the cluster (provision tests only)")
	registerStringFlag(&provision_test.DefaultSettings.CNIBundleURL, "chubo.provision.cni-bundle-url", "talos.provision.cni-bundle-url", provision_test.DefaultSettings.CNIBundleURL, "URL to download CNI bundle from")

	allSuites = slices.Concat(api.GetAllSuites(), cli.GetAllSuites(), provision_test.GetAllSuites())
}
