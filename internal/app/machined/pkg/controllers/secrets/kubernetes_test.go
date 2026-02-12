// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package secrets_test

import (
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/rtestutils"
	"github.com/siderolabs/crypto/x509"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/ctest"
	secretsctrl "github.com/chubo-dev/chubo/internal/app/machined/pkg/controllers/secrets"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
	timeresource "github.com/chubo-dev/chubo/pkg/machinery/resources/time"
)

func TestKubernetesSuite(t *testing.T) {
	suite.Run(t, &KubernetesSuite{
		DefaultSuite: ctest.DefaultSuite{
			Timeout: 5 * time.Second,
			AfterSetup: func(suite *ctest.DefaultSuite) {
				suite.Require().NoError(suite.Runtime().RegisterController(&secretsctrl.KubernetesController{}))
			},
		},
	})
}

type KubernetesSuite struct {
	ctest.DefaultSuite
}

func (suite *KubernetesSuite) TestReconcile() {
	rootSecrets := secrets.NewKubernetesRoot(secrets.KubernetesRootID)

	k8sCA, err := x509.NewSelfSignedCertificateAuthority(
		x509.Organization("kubernetes"),
		x509.ECDSA(true),
	)
	suite.Require().NoError(err)

	aggregatorCA, err := x509.NewSelfSignedCertificateAuthority(
		x509.Organization("kubernetes"),
		x509.ECDSA(true),
	)
	suite.Require().NoError(err)

	serviceAccount, err := x509.NewECDSAKey()
	suite.Require().NoError(err)

	rootSecrets.TypedSpec().Name = "cluster1"
	rootSecrets.TypedSpec().Endpoint, err = url.Parse("https://some.url:6443/")
	suite.Require().NoError(err)
	rootSecrets.TypedSpec().LocalEndpoint, err = url.Parse("https://localhost:6443/")
	suite.Require().NoError(err)

	rootSecrets.TypedSpec().IssuingCA = &x509.PEMEncodedCertificateAndKey{
		Crt: k8sCA.CrtPEM,
		Key: k8sCA.KeyPEM,
	}
	rootSecrets.TypedSpec().AggregatorCA = &x509.PEMEncodedCertificateAndKey{
		Crt: aggregatorCA.CrtPEM,
		Key: aggregatorCA.KeyPEM,
	}
	rootSecrets.TypedSpec().ServiceAccount = &x509.PEMEncodedKey{
		Key: serviceAccount.KeyPEM,
	}
	rootSecrets.TypedSpec().CertSANs = []string{"example.com"}
	rootSecrets.TypedSpec().APIServerIPs = []netip.Addr{netip.MustParseAddr("10.4.3.2"), netip.MustParseAddr("10.2.1.3")}
	rootSecrets.TypedSpec().DNSDomain = "cluster.svc"
	suite.Require().NoError(suite.State().Create(suite.Ctx(), rootSecrets))

	machineType := config.NewMachineType()
	machineType.SetMachineType(machine.TypeControlPlane)
	suite.Require().NoError(suite.State().Create(suite.Ctx(), machineType))

	timeSync := timeresource.NewStatus()
	*timeSync.TypedSpec() = timeresource.StatusSpec{
		Synced: true,
	}
	suite.Require().NoError(suite.State().Create(suite.Ctx(), timeSync))

	rtestutils.AssertResources(suite.Ctx(), suite.T(), suite.State(), []resource.ID{secrets.KubernetesID},
		func(certs *secrets.Kubernetes, assertion *assert.Assertions) {
			kubernetesCerts := certs.TypedSpec()

			for _, kubeconfig := range []string{
				kubernetesCerts.ControllerManagerKubeconfig,
				kubernetesCerts.SchedulerKubeconfig,
				kubernetesCerts.LocalhostAdminKubeconfig,
				kubernetesCerts.AdminKubeconfig,
			} {
				config, err := clientcmd.Load([]byte(kubeconfig))
				assertion.NoError(err)

				if err != nil {
					return
				}

				assertion.NoError(clientcmd.ConfirmUsable(*config, config.CurrentContext))
			}
		})
}
