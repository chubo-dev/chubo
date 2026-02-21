// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package secrets_test

import (
	stdx509 "crypto/x509"
	"testing"
	"time"

	"github.com/siderolabs/crypto/x509"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/chubo-dev/chubo/pkg/machinery/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config/configloader"
	"github.com/chubo-dev/chubo/pkg/machinery/config/generate/secrets"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	v1alpha1 "github.com/chubo-dev/chubo/pkg/machinery/config/types/v1alpha1"
	"github.com/chubo-dev/chubo/pkg/machinery/role"
)

func TestNewBundle(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name            string
		versionContract *config.VersionContract
	}{
		{
			name:            "v1.0",
			versionContract: config.ChuboVersion1_0,
		},
		{
			name:            "v1.3",
			versionContract: config.ChuboVersion1_3,
		},
		{
			name:            "v1.7",
			versionContract: config.ChuboVersion1_7,
		},
		{
			name:            "current",
			versionContract: config.ChuboVersionCurrent,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := secrets.NewBundle(secrets.NewFixedClock(time.Now()), test.versionContract)
			require.NoError(t, err)
		})
	}
}

func TestNewBundleFromConfig(t *testing.T) {
	t.Parallel()

	bundle, err := secrets.NewBundle(secrets.NewFixedClock(time.Now()), config.ChuboVersionCurrent)
	require.NoError(t, err)

	osCA, err := x509.NewCertificateAuthorityFromCertificateAndKey(bundle.Certs.OS)
	require.NoError(t, err)

	assert.Equal(t, stdx509.Ed25519, osCA.Crt.PublicKeyAlgorithm, "expected Ed25519 signature algorithm")

	doc := &v1alpha1.Config{
		ConfigVersion: "v1alpha1",
		MachineConfig: &v1alpha1.MachineConfig{
			MachineType:  machine.TypeControlPlane.String(),
			MachineToken: bundle.TrustdInfo.Token,
			MachineCA:    bundle.Certs.OS,
		},
		ClusterConfig: &v1alpha1.ClusterConfig{
			ClusterID:     bundle.Cluster.ID,
			ClusterSecret: bundle.Cluster.Secret,
			ClusterName:   "test",
		},
	}

	raw, err := yaml.Marshal(doc)
	require.NoError(t, err)

	cfg, err := configloader.NewFromBytes(raw)
	require.NoError(t, err)

	bundle2 := secrets.NewBundleFromConfig(bundle.Clock, cfg)

	require.NotNil(t, bundle2.Certs)
	require.NotNil(t, bundle2.Certs.OS)
	assert.Equal(t, bundle.Certs.OS, bundle2.Certs.OS)

	require.NotNil(t, bundle2.TrustdInfo)
	assert.Equal(t, bundle.TrustdInfo.Token, bundle2.TrustdInfo.Token)

	require.NotNil(t, bundle2.Cluster)
	assert.Equal(t, bundle.Cluster.ID, bundle2.Cluster.ID)
	assert.Equal(t, bundle.Cluster.Secret, bundle2.Cluster.Secret)

	cert, err := bundle2.GenerateChuboAPIClientCertificate(role.MakeSet(role.Admin))
	require.NoError(t, err)
	require.NotEmpty(t, cert.Crt)
	require.NotEmpty(t, cert.Key)
}
