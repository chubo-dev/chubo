// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cluster_test

import (
	"net/netip"
	"testing"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	clusterctrl "github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/cluster"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/ctest"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/cluster"
)

type AffiliateMergeSuite struct {
	ClusterSuite
}

func (suite *AffiliateMergeSuite) TestReconcileDefault() {
	suite.startRuntime()

	suite.Require().NoError(suite.runtime.RegisterController(&clusterctrl.AffiliateMergeController{}))

	affiliate1 := cluster.NewAffiliate(cluster.RawNamespaceName, "k8s/7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC")
	*affiliate1.TypedSpec() = cluster.AffiliateSpec{
		NodeID:       "7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC",
		Hostname:     "foo.com",
		Nodename:     "bar",
		MachineType:  machine.TypeControlPlane,
		Addresses:    []netip.Addr{netip.MustParseAddr("192.168.3.4")},
		ControlPlane: &cluster.ControlPlane{APIServerPort: 6443},
	}

	affiliate2 := cluster.NewAffiliate(cluster.RawNamespaceName, "service/7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC")
	*affiliate2.TypedSpec() = cluster.AffiliateSpec{
		NodeID:      "7x1SuC8Ege5BGXdAfTEff5iQnlWZLfv9h1LGMxA2pYkC",
		Hostname:    "foo.com",
		Nodename:    "bar",
		MachineType: machine.TypeControlPlane,
		Addresses:   []netip.Addr{netip.MustParseAddr("192.168.3.4"), netip.MustParseAddr("10.5.0.2")},
	}

	affiliate3 := cluster.NewAffiliate(cluster.RawNamespaceName, "service/9dwHNUViZlPlIervqX9Qo256RUhrfhgO0xBBnKcKl4F")
	*affiliate3.TypedSpec() = cluster.AffiliateSpec{
		NodeID:      "9dwHNUViZlPlIervqX9Qo256RUhrfhgO0xBBnKcKl4F",
		Hostname:    "worker-1",
		Nodename:    "worker-1",
		MachineType: machine.TypeWorker,
		Addresses:   []netip.Addr{netip.MustParseAddr("192.168.3.5")},
	}

	for _, r := range []resource.Resource{affiliate1, affiliate2, affiliate3} {
		suite.Require().NoError(suite.state.Create(suite.ctx, r))
	}

	// there should be two merged affiliates: one from affiliate1+affiliate2, and another from affiliate3
	ctest.AssertResource(
		suite,
		affiliate1.TypedSpec().NodeID,
		func(r *cluster.Affiliate, asrt *assert.Assertions) {
			spec := r.TypedSpec()

			asrt.Equal(affiliate1.TypedSpec().NodeID, spec.NodeID)
			asrt.Equal([]netip.Addr{netip.MustParseAddr("192.168.3.4"), netip.MustParseAddr("10.5.0.2")}, spec.Addresses)
			asrt.Equal("foo.com", spec.Hostname)
			asrt.Equal("bar", spec.Nodename)
			asrt.Equal(machine.TypeControlPlane, spec.MachineType)
			asrt.Equal(&cluster.ControlPlane{APIServerPort: 6443}, spec.ControlPlane)
		},
	)

	ctest.AssertResource(
		suite,
		affiliate3.TypedSpec().NodeID,
		func(r *cluster.Affiliate, asrt *assert.Assertions) {
			spec := r.TypedSpec()

			asrt.Equal(affiliate3.TypedSpec().NodeID, spec.NodeID)
			asrt.Equal([]netip.Addr{netip.MustParseAddr("192.168.3.5")}, spec.Addresses)
			asrt.Equal("worker-1", spec.Hostname)
			asrt.Equal("worker-1", spec.Nodename)
			asrt.Equal(machine.TypeWorker, spec.MachineType)
			asrt.Nil(spec.ControlPlane)
		},
	)

	// remove the affiliate which provided the control plane metadata, merged affiliate should reflect it
	suite.Require().NoError(suite.state.Destroy(suite.ctx, affiliate1.Metadata()))

	ctest.AssertResource(
		suite,
		affiliate1.TypedSpec().NodeID,
		func(r *cluster.Affiliate, asrt *assert.Assertions) {
			spec := r.TypedSpec()

			asrt.Equal(affiliate1.TypedSpec().NodeID, spec.NodeID)
			asrt.Equal([]netip.Addr{netip.MustParseAddr("192.168.3.4"), netip.MustParseAddr("10.5.0.2")}, spec.Addresses)
			asrt.Equal("foo.com", spec.Hostname)
			asrt.Equal("bar", spec.Nodename)
			asrt.Equal(machine.TypeControlPlane, spec.MachineType)
			asrt.Nil(spec.ControlPlane)
		},
	)

	// remove affiliate3, merged affiliate should be removed
	suite.Require().NoError(suite.state.Destroy(suite.ctx, affiliate3.Metadata()))

	ctest.AssertNoResource[*cluster.Affiliate](suite, affiliate3.TypedSpec().NodeID)
}

func TestAffiliateMergeSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(AffiliateMergeSuite))
}
