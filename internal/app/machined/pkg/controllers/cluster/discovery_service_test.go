// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cluster_test

import (
	"context"
	"crypto/aes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/cosi-project/runtime/pkg/resource/rtestutils"
	"github.com/siderolabs/discovery-api/api/v1alpha1/client/pb"
	"github.com/siderolabs/discovery-client/pkg/client"
	"github.com/siderolabs/go-retry/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"

	clusteradapter "github.com/chubo-dev/chubo/internal/app/machined/pkg/adapters/cluster"
	clusterctrl "github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/cluster"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/ctest"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/proto"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/cluster"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/network"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/runtime"
)

type DiscoveryServiceSuite struct {
	ClusterSuite
}

func (suite *DiscoveryServiceSuite) TestReconcile() {
	suite.startRuntime()

	suite.Require().NoError(suite.runtime.RegisterController(&clusterctrl.DiscoveryServiceController{}))

	serviceEndpoint, err := url.Parse(constants.DefaultDiscoveryServiceEndpoint)
	suite.Require().NoError(err)

	if serviceEndpoint.Port() == "" {
		serviceEndpoint.Host += ":443"
	}

	clusterIDRaw := make([]byte, constants.DefaultClusterIDSize)
	_, err = io.ReadFull(rand.Reader, clusterIDRaw)
	suite.Require().NoError(err)

	clusterID := base64.StdEncoding.EncodeToString(clusterIDRaw)

	encryptionKey := make([]byte, constants.DefaultClusterSecretSize)
	_, err = io.ReadFull(rand.Reader, encryptionKey)
	suite.Require().NoError(err)

	// regular discovery affiliate
	discoveryConfig := cluster.NewConfig(config.NamespaceName, cluster.ConfigID)
	discoveryConfig.TypedSpec().DiscoveryEnabled = true
	discoveryConfig.TypedSpec().RegistryServiceEnabled = true
	discoveryConfig.TypedSpec().ServiceEndpoint = serviceEndpoint.Host
	discoveryConfig.TypedSpec().ServiceClusterID = clusterID
	discoveryConfig.TypedSpec().ServiceEncryptionKey = encryptionKey
	suite.Require().NoError(suite.state.Create(suite.ctx, discoveryConfig))

	nodeIdentity := cluster.NewIdentity(cluster.NamespaceName, cluster.LocalIdentity)
	suite.Require().NoError(clusteradapter.IdentitySpec(nodeIdentity.TypedSpec()).Generate())
	suite.Require().NoError(suite.state.Create(suite.ctx, nodeIdentity))

	localAffiliate := cluster.NewAffiliate(cluster.NamespaceName, nodeIdentity.TypedSpec().NodeID)
	*localAffiliate.TypedSpec() = cluster.AffiliateSpec{
		NodeID:       nodeIdentity.TypedSpec().NodeID,
		Hostname:     "foo.com",
		Nodename:     "bar",
		MachineType:  machine.TypeControlPlane,
		Addresses:    []netip.Addr{netip.MustParseAddr("192.168.3.4")},
		ControlPlane: &cluster.ControlPlane{APIServerPort: 6443},
	}
	suite.Require().NoError(suite.state.Create(suite.ctx, localAffiliate))

	// create a test client connected to the same cluster but under different affiliate ID
	cipher, err := aes.NewCipher(discoveryConfig.TypedSpec().ServiceEncryptionKey)
	suite.Require().NoError(err)

	cli, err := client.NewClient(client.Options{
		Cipher:      cipher,
		Endpoint:    serviceEndpoint.Host,
		ClusterID:   discoveryConfig.TypedSpec().ServiceClusterID,
		AffiliateID: "7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC",
		TTL:         5 * time.Minute,
	})
	suite.Require().NoError(err)

	errCh := make(chan error, 1)
	notifyCh := make(chan struct{}, 1)

	cliCtx, cliCtxCancel := context.WithCancel(suite.ctx)
	defer cliCtxCancel()

	go func() {
		errCh <- cli.Run(cliCtx, zaptest.NewLogger(suite.T()), notifyCh)
	}()

	suite.Assert().NoError(retry.Constant(3*time.Second, retry.WithUnits(100*time.Millisecond)).Retry(
		func() error {
			// controller should register its local affiliate, and we should see it being discovered
			affiliates := cli.GetAffiliates()

			if len(affiliates) != 1 {
				return retry.ExpectedErrorf("affiliates len %d != 1", len(affiliates))
			}

			suite.Require().Len(affiliates[0].Endpoints, 0)
			suite.Assert().True(proto.Equal(&pb.Affiliate{
				NodeId:          nodeIdentity.TypedSpec().NodeID,
				Addresses:       [][]byte{[]byte("\xc0\xa8\x03\x04")},
				Hostname:        "foo.com",
				Nodename:        "bar",
				MachineType:     "controlplane",
				OperatingSystem: "",
				ControlPlane:    &pb.ControlPlane{ApiServerPort: 6443},
			}, affiliates[0].Affiliate))

			return nil
		},
	))

	// inject some affiliate via our client, controller should publish it as an affiliate
	suite.Require().NoError(cli.SetLocalData(&client.Affiliate{
		Affiliate: &pb.Affiliate{
			NodeId:          "7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC",
			Addresses:       [][]byte{[]byte("\xc0\xa8\x03\x05")},
			Hostname:        "some.com",
			Nodename:        "some",
			MachineType:     "worker",
			OperatingSystem: "test OS",
		},
	}, nil))

	ctest.AssertResource(
		suite,
		"service/7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC",
		func(r *cluster.Affiliate, asrt *assert.Assertions) {
			spec := r.TypedSpec()

			suite.Assert().Equal("7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC", spec.NodeID)
			suite.Assert().Equal([]netip.Addr{netip.MustParseAddr("192.168.3.5")}, spec.Addresses)
			suite.Assert().Equal("some.com", spec.Hostname)
			suite.Assert().Equal("some", spec.Nodename)
			suite.Assert().Equal(machine.TypeWorker, spec.MachineType)
			suite.Assert().Equal("test OS", spec.OperatingSystem)
			suite.Assert().Zero(spec.ControlPlane)
		},
		rtestutils.WithNamespace(cluster.RawNamespaceName),
	)

	// controller should publish public IP
	ctest.AssertResource(suite, "service", func(r *network.AddressStatus, assertions *assert.Assertions) {
		spec := r.TypedSpec()

		assertions.True(spec.Address.IsValid())
		assertions.True(spec.Address.IsSingleIP())
	}, rtestutils.WithNamespace(cluster.NamespaceName))

	// pretend that machine is being reset
	machineResetSignal := runtime.NewMachineResetSignal()
	suite.Require().NoError(suite.state.Create(suite.ctx, machineResetSignal))

	// client should see the affiliate being deleted
	suite.Assert().NoError(retry.Constant(3*time.Second, retry.WithUnits(100*time.Millisecond)).Retry(
		func() error {
			// controller should delete its local affiliate
			affiliates := cli.GetAffiliates()

			if len(affiliates) != 0 {
				return retry.ExpectedErrorf("affiliates len %d != 0", len(affiliates))
			}

			return nil
		},
	))

	cliCtxCancel()
	suite.Assert().NoError(<-errCh)
}

func (suite *DiscoveryServiceSuite) TestDisable() {
	suite.startRuntime()

	suite.Require().NoError(suite.runtime.RegisterController(&clusterctrl.DiscoveryServiceController{}))

	serviceEndpoint, err := url.Parse(constants.DefaultDiscoveryServiceEndpoint)
	suite.Require().NoError(err)

	if serviceEndpoint.Port() == "" {
		serviceEndpoint.Host += ":443"
	}

	clusterIDRaw := make([]byte, constants.DefaultClusterIDSize)
	_, err = io.ReadFull(rand.Reader, clusterIDRaw)
	suite.Require().NoError(err)

	clusterID := base64.StdEncoding.EncodeToString(clusterIDRaw)

	encryptionKey := make([]byte, constants.DefaultClusterSecretSize)
	_, err = io.ReadFull(rand.Reader, encryptionKey)
	suite.Require().NoError(err)

	// regular discovery affiliate
	discoveryConfig := cluster.NewConfig(config.NamespaceName, cluster.ConfigID)
	discoveryConfig.TypedSpec().DiscoveryEnabled = true
	discoveryConfig.TypedSpec().RegistryServiceEnabled = true
	discoveryConfig.TypedSpec().ServiceEndpoint = serviceEndpoint.Host
	discoveryConfig.TypedSpec().ServiceClusterID = clusterID
	discoveryConfig.TypedSpec().ServiceEncryptionKey = encryptionKey
	suite.Require().NoError(suite.state.Create(suite.ctx, discoveryConfig))

	nodeIdentity := cluster.NewIdentity(cluster.NamespaceName, cluster.LocalIdentity)
	suite.Require().NoError(clusteradapter.IdentitySpec(nodeIdentity.TypedSpec()).Generate())
	suite.Require().NoError(suite.state.Create(suite.ctx, nodeIdentity))

	localAffiliate := cluster.NewAffiliate(cluster.NamespaceName, nodeIdentity.TypedSpec().NodeID)
	*localAffiliate.TypedSpec() = cluster.AffiliateSpec{
		NodeID:      nodeIdentity.TypedSpec().NodeID,
		Hostname:    "foo.com",
		Nodename:    "bar",
		MachineType: machine.TypeControlPlane,
		Addresses:   []netip.Addr{netip.MustParseAddr("192.168.3.4")},
	}
	suite.Require().NoError(suite.state.Create(suite.ctx, localAffiliate))

	// create a test client connected to the same cluster but under different affiliate ID
	cipher, err := aes.NewCipher(discoveryConfig.TypedSpec().ServiceEncryptionKey)
	suite.Require().NoError(err)

	cli, err := client.NewClient(client.Options{
		Cipher:      cipher,
		Endpoint:    serviceEndpoint.Host,
		ClusterID:   discoveryConfig.TypedSpec().ServiceClusterID,
		AffiliateID: "7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC",
		TTL:         5 * time.Minute,
	})
	suite.Require().NoError(err)

	errCh := make(chan error, 1)
	notifyCh := make(chan struct{}, 1)

	cliCtx, cliCtxCancel := context.WithCancel(suite.ctx)
	defer cliCtxCancel()

	go func() {
		errCh <- cli.Run(cliCtx, zaptest.NewLogger(suite.T()), notifyCh)
	}()

	// inject some affiliate via our client, controller should publish it as an affiliate
	suite.Require().NoError(cli.SetLocalData(&client.Affiliate{
		Affiliate: &pb.Affiliate{
			NodeId: "7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC",
		},
	}, nil))

	ctest.AssertResource(
		suite,
		"service/7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC",
		func(r *cluster.Affiliate, asrt *assert.Assertions) {
			suite.Assert().Equal("7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC", r.TypedSpec().NodeID)
		},
		rtestutils.WithNamespace(cluster.RawNamespaceName),
	)

	// now disable the service registry
	ctest.UpdateWithConflicts(suite, discoveryConfig, func(r *cluster.Config) error {
		r.TypedSpec().RegistryServiceEnabled = false

		return nil
	})

	ctest.AssertNoResource[*cluster.Affiliate](
		suite,
		"service/7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC",
		rtestutils.WithNamespace(cluster.RawNamespaceName),
	)

	cliCtxCancel()
	suite.Assert().NoError(<-errCh)
}

func TestDiscoveryServiceSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(DiscoveryServiceSuite))
}
