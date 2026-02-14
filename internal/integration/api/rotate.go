// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build integration_api

package api

import (
	"context"
	"testing"
	"time"

	"github.com/cosi-project/runtime/pkg/safe"

	"github.com/chubo-dev/chubo/internal/integration/base"
	"github.com/chubo-dev/chubo/pkg/machinery/api/common"
	"github.com/chubo-dev/chubo/pkg/machinery/client"
	"github.com/chubo-dev/chubo/pkg/machinery/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config/encoder"
	"github.com/chubo-dev/chubo/pkg/machinery/config/generate/secrets"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	secretsres "github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
	"github.com/chubo-dev/chubo/pkg/provision/access"
	"github.com/chubo-dev/chubo/pkg/rotate/pki/talos"
)

// RotateCASuite verifies rotation of the OS API CA.
type RotateCASuite struct {
	base.APISuite

	ctx       context.Context //nolint:containedctx
	ctxCancel context.CancelFunc
}

// SuiteName ...
func (suite *RotateCASuite) SuiteName() string {
	return "api.RotateCASuite"
}

// SetupTest ...
func (suite *RotateCASuite) SetupTest() {
	// make sure API calls have timeout
	suite.ctx, suite.ctxCancel = context.WithTimeout(context.Background(), 10*time.Minute)
}

// TearDownTest ...
func (suite *RotateCASuite) TearDownTest() {
	if suite.ctxCancel != nil {
		suite.ctxCancel()
	}
}

// TestTalos updates Talos CA in the cluster.
func (suite *RotateCASuite) TestTalos() {
	if suite.Cluster == nil {
		suite.T().Skip("cluster information is not available")
	}

	suite.T().Logf("capturing current Talos CA")

	nodeInternalIP := suite.RandomDiscoveredNodeInternalIP(machine.TypeControlPlane)

	// save osRoot
	osRoot, err := safe.StateGetByID[*secretsres.OSRoot](client.WithNode(suite.ctx, nodeInternalIP), suite.Client.COSI, secretsres.OSRootID)
	suite.Require().NoError(err)

	suite.T().Logf("rotating current CA -> new CA")

	newBundle, err := secrets.NewBundle(secrets.NewFixedClock(time.Now()), config.TalosVersionCurrent)
	suite.Require().NoError(err)

	options := talos.Options{
		CurrentClient: suite.Client,
		ClusterInfo:   access.NewAdapter(suite.Cluster),

		ContextName: suite.Talosconfig.Context,
		Endpoints:   suite.Client.GetEndpoints(),

		NewTalosCA: newBundle.Certs.OS,

		EncoderOption: encoder.WithComments(encoder.CommentsAll),

		Printf: suite.T().Logf,
	}

	newTalosconfig, err := talos.Rotate(suite.ctx, options)
	suite.Require().NoError(err)

	newClient, err := client.New(suite.ctx, client.WithConfig(newTalosconfig))
	suite.Require().NoError(err)

	if !testing.Short() {
		suite.restartAPIServices(newClient)
	}

	suite.T().Logf("rotating back new CA -> old CA")

	options = talos.Options{
		CurrentClient: newClient,
		ClusterInfo:   access.NewAdapter(suite.Cluster),

		ContextName: suite.Talosconfig.Context,
		Endpoints:   suite.Client.GetEndpoints(),

		NewTalosCA: osRoot.TypedSpec().IssuingCA,

		EncoderOption: encoder.WithComments(encoder.CommentsAll),

		Printf: suite.T().Logf,
	}

	_, err = talos.Rotate(suite.ctx, options)
	suite.Require().NoError(err)

	suite.AssertClusterHealthy(suite.ctx)

	suite.ClearConnectionRefused(suite.ctx, suite.DiscoverNodeInternalIPsByType(suite.ctx, machine.TypeWorker)...)
}

func (suite *RotateCASuite) restartAPIServices(c *client.Client) {
	suite.T().Logf("restarting API services")

	var oldClient *client.Client

	oldClient, suite.Client = suite.Client, c

	defer func() {
		suite.Client = oldClient
	}()

	for _, node := range suite.DiscoverNodeInternalIPsByType(suite.ctx, machine.TypeControlPlane) {
		suite.T().Logf("restarting API services on %s", node)

		err := c.Restart(client.WithNode(suite.ctx, node), constants.SystemContainerdNamespace, common.ContainerDriver_CONTAINERD, "trustd")
		suite.Require().NoError(err)

		suite.ClearConnectionRefused(suite.ctx, node)

		err = c.Restart(client.WithNode(suite.ctx, node), constants.SystemContainerdNamespace, common.ContainerDriver_CONTAINERD, "apid")
		suite.Require().NoError(err)

		suite.ClearConnectionRefused(suite.ctx, node)
	}

	for _, node := range suite.DiscoverNodeInternalIPsByType(suite.ctx, machine.TypeWorker) {
		suite.T().Logf("restarting API services on %s", node)

		err := c.Restart(client.WithNode(suite.ctx, node), constants.SystemContainerdNamespace, common.ContainerDriver_CONTAINERD, "apid")
		suite.Require().NoError(err)

		suite.ClearConnectionRefused(suite.ctx, node)
	}

	suite.AssertClusterHealthy(suite.ctx)
}

func init() {
	allSuites = append(allSuites, new(RotateCASuite))
}
