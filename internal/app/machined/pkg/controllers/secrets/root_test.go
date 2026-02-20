// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package secrets_test

import (
	"testing"
	"time"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/rtestutils"
	"github.com/siderolabs/crypto/x509"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.yaml.in/yaml/v4"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/ctest"
	secretsctrl "github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/secrets"
	talosconfig "github.com/chubo-dev/chubo/pkg/machinery/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config/configloader"
	gensecrets "github.com/chubo-dev/chubo/pkg/machinery/config/generate/secrets"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	v1alpha1 "github.com/chubo-dev/chubo/pkg/machinery/config/types/v1alpha1"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
)

func TestRootSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, &RootSuite{
		DefaultSuite: ctest.DefaultSuite{
			Timeout: 10 * time.Second,
			AfterSetup: func(suite *ctest.DefaultSuite) {
				suite.Require().NoError(suite.Runtime().RegisterController(secretsctrl.NewRootOSController()))
			},
		},
	})
}

type RootSuite struct {
	ctest.DefaultSuite
}

func (suite *RootSuite) genConfig(controlplane bool) talosconfig.Provider {
	bundle, err := gensecrets.NewBundle(gensecrets.NewFixedClock(time.Now()), talosconfig.ChuboVersionCurrent)
	suite.Require().NoError(err)

	nodeType := machine.TypeWorker
	nodeCA := &x509.PEMEncodedCertificateAndKey{Crt: bundle.Certs.OS.Crt}

	if controlplane {
		nodeType = machine.TypeControlPlane
		nodeCA = bundle.Certs.OS
	}

	cfgDoc := &v1alpha1.Config{
		ConfigVersion: "v1alpha1",
		MachineConfig: &v1alpha1.MachineConfig{
			MachineType:  nodeType.String(),
			MachineToken: bundle.TrustdInfo.Token,
			MachineCA:    nodeCA,
		},
		ClusterConfig: &v1alpha1.ClusterConfig{
			ClusterID:     bundle.Cluster.ID,
			ClusterSecret: bundle.Cluster.Secret,
			ClusterName:   "test-cluster",
		},
	}

	raw, err := yaml.Marshal(cfgDoc)
	suite.Require().NoError(err)

	cfg, err := configloader.NewFromBytes(raw)
	suite.Require().NoError(err)

	machineCfg := config.NewMachineConfig(cfg)
	suite.Require().NoError(suite.State().Create(suite.Ctx(), machineCfg))

	return cfg
}

func (suite *RootSuite) TestReconcileControlPlane() {
	cfg := suite.genConfig(true)

	rtestutils.AssertResources(suite.Ctx(), suite.T(), suite.State(), []resource.ID{secrets.OSRootID},
		func(res *secrets.OSRoot, asrt *assert.Assertions) {
			asrt.Equal(res.TypedSpec().IssuingCA, cfg.Machine().Security().IssuingCA())
			asrt.Equal(
				[]*x509.PEMEncodedCertificate{
					{
						Crt: cfg.Machine().Security().IssuingCA().Crt,
					},
				},
				res.TypedSpec().AcceptedCAs,
			)
		},
	)
}

func (suite *RootSuite) TestReconcileWorker() {
	cfg := suite.genConfig(false)

	rtestutils.AssertResources(suite.Ctx(), suite.T(), suite.State(), []resource.ID{secrets.OSRootID},
		func(res *secrets.OSRoot, asrt *assert.Assertions) {
			asrt.Nil(res.TypedSpec().IssuingCA)
			asrt.Equal(
				[]*x509.PEMEncodedCertificate{
					{
						Crt: cfg.Machine().Security().IssuingCA().Crt,
					},
				},
				res.TypedSpec().AcceptedCAs,
			)
		},
	)
}
