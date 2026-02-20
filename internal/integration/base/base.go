// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build integration

// Package base provides shared definition of base suites for tests
package base

import (
	"context"

	"github.com/chubo-dev/chubo/pkg/cluster"
	"github.com/chubo-dev/chubo/pkg/provision"
	"github.com/chubo-dev/chubo/pkg/provision/access"
)

const (
	// ProvisionerQEMU is the name of the QEMU provisioner.
	ProvisionerQEMU = "qemu"
	// ProvisionerDocker is the name of the Docker provisioner.
	ProvisionerDocker = "docker"
)

// ChuboSuite defines most common settings for integration test suites.
type ChuboSuite struct {
	// Endpoint to use to connect, if not set config is used
	Endpoint string
	// K8sEndpoint is API server endpoint, if set overrides kubeconfig
	K8sEndpoint string
	// Cluster describes provisioned cluster, used for discovery purposes
	Cluster provision.Cluster
	// ChuboconfigPath is a path to chuboconfig.
	ChuboconfigPath string
	// Version is the expected OS version tests are running against.
	Version string
	// GoVersion is the (expected) version of Go compiler.
	GoVersion string
	// ChuboctlPath is a path to chuboctl binary.
	ChuboctlPath string
	// KubectlPath is a path to kubectl binary
	KubectlPath string
	// HelmPath is a path to helm binary
	HelmPath string
	// KubeStrPath is a path to kubestr binary
	KubeStrPath string
	// ExtensionsQEMU runs tests with qemu and extensions enabled
	ExtensionsQEMU bool
	// ExtensionsNvidia runs tests with nvidia extensions enabled
	ExtensionsNvidia bool
	// TrustedBoot tells if the cluster is secure booted and disks are encrypted
	TrustedBoot bool
	// SelinuxEnforcing tells if the cluster is booted with the image with selinux enforcement enabled
	SelinuxEnforcing bool
	// VerifyUKIBooted runs tests to verify the node is booted from a UKI.
	VerifyUKIBooted bool
	// ChuboImage is the image name for the OS container image.
	ChuboImage string
	// CSITestName is the name of the CSI test to run
	CSITestName string
	// CSITestTimeout is the timeout for the CSI test
	CSITestTimeout string
	// Airgapped marks that cluster has no access to external networks
	Airgapped bool
	// Virtiofsd marks that cluster has virtiofs volumes (virtiofsd is running for workers)
	Virtiofsd bool
	// Race informs test suites about race detector being enabled (e.g. for skipping incompatible tests)
	Race bool

	discoveredNodes cluster.Info
}

// DiscoverNodes provides basic functionality to discover cluster nodes via test settings.
//
// This method is overridden in specific suites to allow for specific discovery.
func (chuboSuite *ChuboSuite) DiscoverNodes(_ context.Context) cluster.Info {
	if chuboSuite.discoveredNodes == nil {
		if chuboSuite.Cluster != nil {
			chuboSuite.discoveredNodes = access.NewAdapter(chuboSuite.Cluster).Info
		}
	}

	return chuboSuite.discoveredNodes
}

// ConfiguredSuite expects config to be set before running.
type ConfiguredSuite interface {
	SetConfig(config ChuboSuite)
}

// SetConfig implements ConfiguredSuite.
func (chuboSuite *ChuboSuite) SetConfig(config ChuboSuite) {
	*chuboSuite = config
}

// NamedSuite interface provides names for test suites.
type NamedSuite interface {
	SuiteName() string
}
