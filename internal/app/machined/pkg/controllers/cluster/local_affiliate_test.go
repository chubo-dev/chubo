// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cluster_test

import (
	"net/netip"
	"testing"

	"github.com/siderolabs/gen/xslices"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	clusteradapter "github.com/chubo-dev/chubo/internal/app/machined/pkg/adapters/cluster"
	clusterctrl "github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/cluster"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/ctest"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/cluster"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/network"
	"github.com/chubo-dev/chubo/pkg/machinery/version"
)

type LocalAffiliateSuite struct {
	ClusterSuite
}

func (suite *LocalAffiliateSuite) TestGeneration() {
	suite.startRuntime()

	suite.Require().NoError(suite.runtime.RegisterController(&clusterctrl.LocalAffiliateController{}))

	nodeIdentity, _, discoveryConfig := suite.createResources()

	machineType := config.NewMachineType()
	machineType.SetMachineType(machine.TypeWorker)
	suite.Require().NoError(suite.state.Create(suite.ctx, machineType))

	ctest.AssertResource(suite, nodeIdentity.TypedSpec().NodeID, func(r *cluster.Affiliate, asrt *assert.Assertions) {
		spec := r.TypedSpec()

		asrt.Equal([]string{
			"172.20.0.2",
			"10.5.0.1",
			"192.168.192.168",
			"2001:123:4567::1",
		}, xslices.Map(spec.Addresses, netip.Addr.String))
		asrt.Equal("example1.com", spec.Hostname)
		asrt.Equal("example1.com", spec.Nodename)
		asrt.Equal(machine.TypeWorker, spec.MachineType)
		asrt.Equal(version.Name+" ("+version.Tag+")", spec.OperatingSystem)
		asrt.Nil(spec.ControlPlane)
	})

	// disable discovery, local affiliate should be removed
	discoveryConfig.TypedSpec().DiscoveryEnabled = false
	suite.Require().NoError(suite.state.Update(suite.ctx, discoveryConfig))

	ctest.AssertNoResource[*cluster.Affiliate](suite, nodeIdentity.TypedSpec().NodeID)
}

func (suite *LocalAffiliateSuite) TestCPGeneration() {
	suite.startRuntime()

	suite.Require().NoError(suite.runtime.RegisterController(&clusterctrl.LocalAffiliateController{}))

	nodeIdentity, _, discoveryConfig := suite.createResources()

	machineType := config.NewMachineType()
	machineType.SetMachineType(machine.TypeControlPlane)
	suite.Require().NoError(suite.state.Create(suite.ctx, machineType))

	ctest.AssertResource(suite, nodeIdentity.TypedSpec().NodeID, func(r *cluster.Affiliate, asrt *assert.Assertions) {
		spec := r.TypedSpec()

		asrt.Equal([]string{
			"172.20.0.2",
			"10.5.0.1",
			"192.168.192.168",
			"2001:123:4567::1",
		}, xslices.Map(spec.Addresses, netip.Addr.String))
		asrt.Equal("example1.com", spec.Hostname)
		asrt.Equal("example1.com", spec.Nodename)
		asrt.Equal(machine.TypeControlPlane, spec.MachineType)
		asrt.Equal(version.Name+" ("+version.Tag+")", spec.OperatingSystem)
		asrt.Nil(spec.ControlPlane)
	})

	discoveryConfig.TypedSpec().DiscoveryEnabled = false
	suite.Require().NoError(suite.state.Update(suite.ctx, discoveryConfig))

	ctest.AssertNoResource[*cluster.Affiliate](suite, nodeIdentity.TypedSpec().NodeID)
}

func (suite *LocalAffiliateSuite) createResources() (*cluster.Identity, *network.NodeAddress, *cluster.Config) {
	// regular discovery affiliate
	discoveryConfig := cluster.NewConfig(config.NamespaceName, cluster.ConfigID)
	discoveryConfig.TypedSpec().DiscoveryEnabled = true
	suite.Require().NoError(suite.state.Create(suite.ctx, discoveryConfig))

	nodeIdentity := cluster.NewIdentity(cluster.NamespaceName, cluster.LocalIdentity)
	suite.Require().NoError(clusteradapter.IdentitySpec(nodeIdentity.TypedSpec()).Generate())
	suite.Require().NoError(suite.state.Create(suite.ctx, nodeIdentity))

	hostnameStatus := network.NewHostnameStatus(network.NamespaceName, network.HostnameID)
	hostnameStatus.TypedSpec().Hostname = "example1"
	hostnameStatus.TypedSpec().Domainname = "com"
	suite.Require().NoError(suite.state.Create(suite.ctx, hostnameStatus))

	nonK8sCurrentAddresses := network.NewNodeAddress(network.NamespaceName, network.NodeAddressCurrentID)
	nonK8sCurrentAddresses.TypedSpec().Addresses = []netip.Prefix{
		netip.MustParsePrefix("172.20.0.2/24"),
		netip.MustParsePrefix("10.5.0.1/32"),
		netip.MustParsePrefix("192.168.192.168/24"),
		netip.MustParsePrefix("2001:123:4567::1/64"),
		netip.MustParsePrefix("2001:123:4567::1/128"),
		netip.MustParsePrefix("fdae:41e4:649b:9303:60be:7e36:c270:3238/128"), // SideroLink, should be ignored
	}
	suite.Require().NoError(suite.state.Create(suite.ctx, nonK8sCurrentAddresses))

	nonK8sRoutedAddresses := network.NewNodeAddress(network.NamespaceName, network.NodeAddressRoutedID)
	nonK8sRoutedAddresses.TypedSpec().Addresses = []netip.Prefix{ // routed node addresses don't contain SideroLink addresses
		netip.MustParsePrefix("172.20.0.2/24"),
		netip.MustParsePrefix("10.5.0.1/32"),
		netip.MustParsePrefix("192.168.192.168/24"),
		netip.MustParsePrefix("2001:123:4567::1/64"),
		netip.MustParsePrefix("2001:123:4567::1/128"),
	}
	suite.Require().NoError(suite.state.Create(suite.ctx, nonK8sRoutedAddresses))

	return nodeIdentity, nonK8sRoutedAddresses, discoveryConfig
}

func TestLocalAffiliateSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(LocalAffiliateSuite))
}
