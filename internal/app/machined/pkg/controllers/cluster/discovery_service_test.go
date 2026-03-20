// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cluster_test

import (
	"context"
	"crypto/aes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/netip"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/cosi-project/runtime/pkg/resource/rtestutils"
	clientpb "github.com/siderolabs/discovery-api/api/v1alpha1/client/pb"
	serverpb "github.com/siderolabs/discovery-api/api/v1alpha1/server/pb"
	"github.com/siderolabs/discovery-client/pkg/client"
	"github.com/siderolabs/go-retry/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"

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

	discoveryService := newLocalDiscoveryService(suite.T())

	clusterID, encryptionKey := newDiscoveryClusterCredentials(suite.T())

	discoveryConfig := cluster.NewConfig(config.NamespaceName, cluster.ConfigID)
	discoveryConfig.TypedSpec().DiscoveryEnabled = true
	discoveryConfig.TypedSpec().RegistryServiceEnabled = true
	discoveryConfig.TypedSpec().ServiceEndpoint = discoveryService.Endpoint()
	discoveryConfig.TypedSpec().ServiceEndpointInsecure = true
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

	cli := newDiscoveryClient(suite.T(), discoveryConfig, "7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC")

	errCh := make(chan error, 1)
	notifyCh := make(chan struct{}, 1)

	cliCtx, cliCtxCancel := context.WithCancel(suite.ctx)
	defer cliCtxCancel()

	go func() {
		errCh <- cli.Run(cliCtx, zaptest.NewLogger(suite.T()), notifyCh)
	}()

	suite.Assert().NoError(retry.Constant(3*time.Second, retry.WithUnits(100*time.Millisecond)).Retry(
		func() error {
			affiliates := cli.GetAffiliates()

			if len(affiliates) != 1 {
				return retry.ExpectedErrorf("affiliates len %d != 1", len(affiliates))
			}

			suite.Require().Len(affiliates[0].Endpoints, 0)
			suite.Assert().True(proto.Equal(&clientpb.Affiliate{
				NodeId:          nodeIdentity.TypedSpec().NodeID,
				Addresses:       [][]byte{[]byte("\xc0\xa8\x03\x04")},
				Hostname:        "foo.com",
				Nodename:        "bar",
				MachineType:     "controlplane",
				OperatingSystem: "",
				ControlPlane:    &clientpb.ControlPlane{ApiServerPort: 6443},
			}, affiliates[0].Affiliate))

			return nil
		},
	))

	suite.Require().NoError(cli.SetLocalData(&client.Affiliate{
		Affiliate: &clientpb.Affiliate{
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

	ctest.AssertResource(suite, "service", func(r *network.AddressStatus, assertions *assert.Assertions) {
		spec := r.TypedSpec()

		assertions.True(spec.Address.IsValid())
		assertions.True(spec.Address.IsSingleIP())
		assertions.Equal(netip.MustParseAddr("127.0.0.1"), spec.Address.Addr())
	}, rtestutils.WithNamespace(cluster.NamespaceName))

	machineResetSignal := runtime.NewMachineResetSignal()
	suite.Require().NoError(suite.state.Create(suite.ctx, machineResetSignal))

	suite.Assert().NoError(retry.Constant(3*time.Second, retry.WithUnits(100*time.Millisecond)).Retry(
		func() error {
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

	discoveryService := newLocalDiscoveryService(suite.T())

	clusterID, encryptionKey := newDiscoveryClusterCredentials(suite.T())

	discoveryConfig := cluster.NewConfig(config.NamespaceName, cluster.ConfigID)
	discoveryConfig.TypedSpec().DiscoveryEnabled = true
	discoveryConfig.TypedSpec().RegistryServiceEnabled = true
	discoveryConfig.TypedSpec().ServiceEndpoint = discoveryService.Endpoint()
	discoveryConfig.TypedSpec().ServiceEndpointInsecure = true
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

	cli := newDiscoveryClient(suite.T(), discoveryConfig, "7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC")

	errCh := make(chan error, 1)
	notifyCh := make(chan struct{}, 1)

	cliCtx, cliCtxCancel := context.WithCancel(suite.ctx)
	defer cliCtxCancel()

	go func() {
		errCh <- cli.Run(cliCtx, zaptest.NewLogger(suite.T()), notifyCh)
	}()

	suite.Require().NoError(cli.SetLocalData(&client.Affiliate{
		Affiliate: &clientpb.Affiliate{
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

func newDiscoveryClusterCredentials(t testing.TB) (string, []byte) {
	t.Helper()

	clusterIDRaw := make([]byte, constants.DefaultClusterIDSize)
	_, err := io.ReadFull(rand.Reader, clusterIDRaw)
	require.NoError(t, err)

	encryptionKey := make([]byte, constants.DefaultClusterSecretSize)
	_, err = io.ReadFull(rand.Reader, encryptionKey)
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(clusterIDRaw), encryptionKey
}

func newDiscoveryClient(t testing.TB, discoveryConfig *cluster.Config, affiliateID string) *client.Client {
	t.Helper()

	cipher, err := aes.NewCipher(discoveryConfig.TypedSpec().ServiceEncryptionKey)
	require.NoError(t, err)

	cli, err := client.NewClient(client.Options{
		Cipher:      cipher,
		Endpoint:    discoveryConfig.TypedSpec().ServiceEndpoint,
		ClusterID:   discoveryConfig.TypedSpec().ServiceClusterID,
		AffiliateID: affiliateID,
		TTL:         5 * time.Minute,
		Insecure:    discoveryConfig.TypedSpec().ServiceEndpointInsecure,
	})
	require.NoError(t, err)

	return cli
}

type localDiscoveryService struct {
	serverpb.UnimplementedClusterServer

	listener   net.Listener
	grpcServer *grpc.Server
	serveErrCh chan error

	mu         sync.Mutex
	affiliates map[string]map[string]*serverpb.Affiliate
	watchers   map[string]map[int]chan *serverpb.WatchResponse
	nextID     int
}

func newLocalDiscoveryService(t testing.TB) *localDiscoveryService {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	service := &localDiscoveryService{
		listener:   listener,
		grpcServer: grpc.NewServer(),
		serveErrCh: make(chan error, 1),
		affiliates: map[string]map[string]*serverpb.Affiliate{},
		watchers:   map[string]map[int]chan *serverpb.WatchResponse{},
	}

	serverpb.RegisterClusterServer(service.grpcServer, service)

	go func() {
		service.serveErrCh <- service.grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		service.grpcServer.Stop()

		err := <-service.serveErrCh
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			require.NoError(t, err)
		}
	})

	return service
}

func (service *localDiscoveryService) Endpoint() string {
	return service.listener.Addr().String()
}

func (service *localDiscoveryService) Hello(context.Context, *serverpb.HelloRequest) (*serverpb.HelloResponse, error) {
	return &serverpb.HelloResponse{
		ClientIp: netip.MustParseAddr("127.0.0.1").AsSlice(),
	}, nil
}

func (service *localDiscoveryService) AffiliateUpdate(_ context.Context, req *serverpb.AffiliateUpdateRequest) (*serverpb.AffiliateUpdateResponse, error) {
	updated := service.upsertAffiliate(req)
	service.broadcast(req.GetClusterId(), &serverpb.WatchResponse{Affiliates: []*serverpb.Affiliate{updated}})

	return &serverpb.AffiliateUpdateResponse{}, nil
}

func (service *localDiscoveryService) AffiliateDelete(_ context.Context, req *serverpb.AffiliateDeleteRequest) (*serverpb.AffiliateDeleteResponse, error) {
	service.deleteAffiliate(req.GetClusterId(), req.GetAffiliateId())
	service.broadcast(req.GetClusterId(), &serverpb.WatchResponse{
		Affiliates: []*serverpb.Affiliate{{Id: req.GetAffiliateId()}},
		Deleted:    true,
	})

	return &serverpb.AffiliateDeleteResponse{}, nil
}

func (service *localDiscoveryService) List(_ context.Context, req *serverpb.ListRequest) (*serverpb.ListResponse, error) {
	return &serverpb.ListResponse{
		Affiliates: service.snapshot(req.GetClusterId()),
	}, nil
}

func (service *localDiscoveryService) Watch(req *serverpb.WatchRequest, stream grpc.ServerStreamingServer[serverpb.WatchResponse]) error {
	clusterID := req.GetClusterId()
	updateCh, watcherID := service.addWatcher(clusterID)
	defer service.removeWatcher(clusterID, watcherID)

	if err := stream.Send(&serverpb.WatchResponse{Affiliates: service.snapshot(clusterID)}); err != nil {
		return err
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case update := <-updateCh:
			if err := stream.Send(update); err != nil {
				return err
			}
		}
	}
}

func (service *localDiscoveryService) upsertAffiliate(req *serverpb.AffiliateUpdateRequest) *serverpb.Affiliate {
	service.mu.Lock()
	defer service.mu.Unlock()

	clusterAffiliates := service.affiliates[req.GetClusterId()]
	if clusterAffiliates == nil {
		clusterAffiliates = map[string]*serverpb.Affiliate{}
		service.affiliates[req.GetClusterId()] = clusterAffiliates
	}

	affiliate := cloneServerAffiliate(clusterAffiliates[req.GetAffiliateId()])
	if affiliate == nil {
		affiliate = &serverpb.Affiliate{Id: req.GetAffiliateId()}
	}

	if data := req.GetAffiliateData(); len(data) > 0 {
		affiliate.Data = cloneBytes(data)
	}

	if endpoints := req.GetAffiliateEndpoints(); len(endpoints) > 0 {
		affiliate.Endpoints = cloneBytes2(endpoints)
	}

	clusterAffiliates[req.GetAffiliateId()] = affiliate

	return cloneServerAffiliate(affiliate)
}

func (service *localDiscoveryService) deleteAffiliate(clusterID, affiliateID string) {
	service.mu.Lock()
	defer service.mu.Unlock()

	clusterAffiliates := service.affiliates[clusterID]
	if clusterAffiliates == nil {
		return
	}

	delete(clusterAffiliates, affiliateID)
}

func (service *localDiscoveryService) snapshot(clusterID string) []*serverpb.Affiliate {
	service.mu.Lock()
	defer service.mu.Unlock()

	clusterAffiliates := service.affiliates[clusterID]
	ids := make([]string, 0, len(clusterAffiliates))

	for affiliateID := range clusterAffiliates {
		ids = append(ids, affiliateID)
	}

	sort.Strings(ids)

	snapshot := make([]*serverpb.Affiliate, 0, len(ids))

	for _, affiliateID := range ids {
		snapshot = append(snapshot, cloneServerAffiliate(clusterAffiliates[affiliateID]))
	}

	return snapshot
}

func (service *localDiscoveryService) addWatcher(clusterID string) (chan *serverpb.WatchResponse, int) {
	service.mu.Lock()
	defer service.mu.Unlock()

	clusterWatchers := service.watchers[clusterID]
	if clusterWatchers == nil {
		clusterWatchers = map[int]chan *serverpb.WatchResponse{}
		service.watchers[clusterID] = clusterWatchers
	}

	watcherID := service.nextID
	service.nextID++

	updateCh := make(chan *serverpb.WatchResponse, 16)
	clusterWatchers[watcherID] = updateCh

	return updateCh, watcherID
}

func (service *localDiscoveryService) removeWatcher(clusterID string, watcherID int) {
	service.mu.Lock()
	defer service.mu.Unlock()

	clusterWatchers := service.watchers[clusterID]
	if clusterWatchers == nil {
		return
	}

	delete(clusterWatchers, watcherID)

	if len(clusterWatchers) == 0 {
		delete(service.watchers, clusterID)
	}
}

func (service *localDiscoveryService) broadcast(clusterID string, update *serverpb.WatchResponse) {
	service.mu.Lock()
	clusterWatchers := service.watchers[clusterID]
	watchers := make([]chan *serverpb.WatchResponse, 0, len(clusterWatchers))

	for _, updateCh := range clusterWatchers {
		watchers = append(watchers, updateCh)
	}

	service.mu.Unlock()

	for _, updateCh := range watchers {
		updateCh <- cloneWatchResponse(update)
	}
}

func cloneServerAffiliate(affiliate *serverpb.Affiliate) *serverpb.Affiliate {
	if affiliate == nil {
		return nil
	}

	return &serverpb.Affiliate{
		Id:        affiliate.Id,
		Data:      cloneBytes(affiliate.Data),
		Endpoints: cloneBytes2(affiliate.Endpoints),
	}
}

func cloneWatchResponse(response *serverpb.WatchResponse) *serverpb.WatchResponse {
	if response == nil {
		return nil
	}

	affiliates := make([]*serverpb.Affiliate, 0, len(response.Affiliates))
	for _, affiliate := range response.Affiliates {
		affiliates = append(affiliates, cloneServerAffiliate(affiliate))
	}

	return &serverpb.WatchResponse{
		Affiliates: affiliates,
		Deleted:    response.Deleted,
	}
}

func cloneBytes(data []byte) []byte {
	if data == nil {
		return nil
	}

	cloned := make([]byte, len(data))
	copy(cloned, data)

	return cloned
}

func cloneBytes2(items [][]byte) [][]byte {
	if items == nil {
		return nil
	}

	cloned := make([][]byte, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, cloneBytes(item))
	}

	return cloned
}
