// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package network_test

import (
	"testing"
	"time"

	"github.com/cosi-project/runtime/pkg/resource/rtestutils"
	"github.com/siderolabs/go-procfs/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/ctest"
	netctrl "github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/network"
	"github.com/chubo-dev/chubo/pkg/machinery/config/container"
	networkcfg "github.com/chubo-dev/chubo/pkg/machinery/config/types/network"
	"github.com/chubo-dev/chubo/pkg/machinery/config/types/v1alpha1"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/network"
)

type TimeServerConfigSuite struct {
	ctest.DefaultSuite
}

func (suite *TimeServerConfigSuite) TestDefaults() {
	suite.Require().NoError(suite.Runtime().RegisterController(&netctrl.TimeServerConfigController{}))

	ctest.AssertResources(
		suite,
		[]string{
			"default/timeservers",
		}, func(r *network.TimeServerSpec, asrt *assert.Assertions) {
			asrt.Equal([]string{constants.DefaultNTPServer}, r.TypedSpec().NTPServers)
			asrt.Equal(network.ConfigDefault, r.TypedSpec().ConfigLayer)
		},
		rtestutils.WithNamespace(network.ConfigNamespaceName),
	)
}

func (suite *TimeServerConfigSuite) TestCmdline() {
	suite.Require().NoError(
		suite.Runtime().RegisterController(
			&netctrl.TimeServerConfigController{
				Cmdline: procfs.NewCmdline("ip=172.20.0.2:172.21.0.1:172.20.0.1:255.255.255.0:master1:eth1::10.0.0.1:10.0.0.2:10.0.0.1"),
			},
		),
	)

	ctest.AssertResources(
		suite,
		[]string{
			"cmdline/timeservers",
		}, func(r *network.TimeServerSpec, asrt *assert.Assertions) {
			asrt.Equal([]string{"10.0.0.1"}, r.TypedSpec().NTPServers)
		},
		rtestutils.WithNamespace(network.ConfigNamespaceName),
	)
}

func (suite *TimeServerConfigSuite) TestMachineConfigurationLegacy() {
	suite.Require().NoError(suite.Runtime().RegisterController(&netctrl.TimeServerConfigController{}))

	cfg := config.NewMachineConfig(
		container.NewV1Alpha1(
			&v1alpha1.Config{
				ConfigVersion: "v1alpha1",
				MachineConfig: &v1alpha1.MachineConfig{
					MachineTime: &v1alpha1.TimeConfig{
						TimeServers: []string{"za.pool.ntp.org", "pool.ntp.org"},
					},
				},
				ClusterConfig: &v1alpha1.ClusterConfig{},
			},
		),
	)

	suite.Create(cfg)

	ctest.AssertResources(
		suite,
		[]string{
			"configuration/timeservers",
		}, func(r *network.TimeServerSpec, asrt *assert.Assertions) {
			asrt.Equal([]string{"za.pool.ntp.org", "pool.ntp.org"}, r.TypedSpec().NTPServers)
		},
		rtestutils.WithNamespace(network.ConfigNamespaceName),
	)

	ctest.UpdateWithConflicts(suite, cfg, func(r *config.MachineConfig) error {
		r.Container().RawV1Alpha1().MachineConfig.MachineTime = nil //nolint:staticcheck

		return nil
	})

	ctest.AssertNoResource[*network.TimeServerSpec](suite, "configuration/timeservers", rtestutils.WithNamespace(network.ConfigNamespaceName))
}

func (suite *TimeServerConfigSuite) TestMachineConfigurationNewStyle() {
	suite.Require().NoError(suite.Runtime().RegisterController(&netctrl.TimeServerConfigController{}))

	tsc := networkcfg.NewTimeSyncConfigV1Alpha1()
	tsc.TimeNTP = &networkcfg.NTPConfig{
		Servers: []string{"za.pool.ntp.org", "pool.ntp.org"},
	}

	ctr, err := container.New(tsc)
	suite.Require().NoError(err)

	cfg := config.NewMachineConfig(ctr)
	suite.Create(cfg)

	ctest.AssertResources(
		suite,
		[]string{
			"configuration/timeservers",
		}, func(r *network.TimeServerSpec, asrt *assert.Assertions) {
			asrt.Equal([]string{"za.pool.ntp.org", "pool.ntp.org"}, r.TypedSpec().NTPServers)
		},
		rtestutils.WithNamespace(network.ConfigNamespaceName),
	)

	suite.Destroy(cfg)

	ctest.AssertNoResource[*network.TimeServerSpec](suite, "configuration/timeservers", rtestutils.WithNamespace(network.ConfigNamespaceName))
}

func TestTimeServerConfigSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, &TimeServerConfigSuite{
		DefaultSuite: ctest.DefaultSuite{
			Timeout: 10 * time.Second,
		},
	})
}
