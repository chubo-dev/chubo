// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chuboos

import (
	"crypto/ed25519"
	"crypto/rand"
	stdlibx509 "crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/siderolabs/talos/pkg/machinery/config/validation"
)

type testRuntimeMode struct{}

func (testRuntimeMode) String() string        { return "test" }
func (testRuntimeMode) RequiresInstall() bool { return false }
func (testRuntimeMode) InContainer() bool     { return false }

func TestMachineConfigBootstrapSignedPayload(t *testing.T) {
	t.Parallel()

	signerCertPEM, jws, payload := genEd25519SignerAndJWS(t, []byte(`{"hello":"world"}`))

	mc := NewMachineConfigV1Alpha1()
	mc.Spec.Trust = &TrustSpec{
		Token: "token",
		CA: &CASpec{
			Crt: "dummy-ca-crt",
			Key: "dummy-ca-key",
		},
	}

	mc.Spec.Modules = &ModulesSpec{
		Chubo: &ChuboModuleSpec{
			Bootstrap: &ChuboBootstrapSpec{
				Mode:       ChuboBootstrapModeSignedPayload,
				SignerCert: signerCertPEM,
				Payload:    jws,
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	var (
		foundPayload bool
		foundFP      bool
	)

	for _, f := range cfg.MachineConfig.MachineFiles {
		switch f.FilePath {
		case "/var/lib/chubo/bootstrap/bootstrap.json":
			foundPayload = true
			require.Equal(t, string(payload), f.FileContent)
			require.Equal(t, "create", f.FileOp)
		case "/var/lib/chubo/bootstrap/signer.sha256":
			foundFP = true
		}
	}

	require.True(t, foundPayload)
	require.True(t, foundFP)
}

func TestMachineConfigBootstrapDisabledModuleErrors(t *testing.T) {
	t.Parallel()

	signerCertPEM, jws, _ := genEd25519SignerAndJWS(t, []byte(`{"hello":"world"}`))

	enabled := false

	mc := NewMachineConfigV1Alpha1()
	mc.Spec.Trust = &TrustSpec{
		Token: "token",
		CA: &CASpec{
			Crt: "dummy-ca-crt",
			Key: "dummy-ca-key",
		},
	}

	mc.Spec.Modules = &ModulesSpec{
		Chubo: &ChuboModuleSpec{
			Enabled: &enabled,
			Bootstrap: &ChuboBootstrapSpec{
				Mode:       ChuboBootstrapModeSignedPayload,
				SignerCert: signerCertPEM,
				Payload:    jws,
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec.modules.chubo.bootstrap is set but spec.modules.chubo.enabled is false")
}

func TestMachineConfigBootstrapBadSignatureErrors(t *testing.T) {
	t.Parallel()

	signerCertPEM, jws, _ := genEd25519SignerAndJWS(t, []byte(`{"hello":"world"}`))
	jws = jws + "x" // corrupt

	mc := NewMachineConfigV1Alpha1()
	mc.Spec.Trust = &TrustSpec{
		Token: "token",
		CA: &CASpec{
			Crt: "dummy-ca-crt",
			Key: "dummy-ca-key",
		},
	}

	mc.Spec.Modules = &ModulesSpec{
		Chubo: &ChuboModuleSpec{
			Bootstrap: &ChuboBootstrapSpec{
				Mode:       ChuboBootstrapModeSignedPayload,
				SignerCert: signerCertPEM,
				Payload:    jws,
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid chubo bootstrap payload")
}

func genEd25519SignerAndJWS(t *testing.T, payload []byte) (signerCertPEM string, jws string, decodedPayload []byte) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	der := newSelfSignedEd25519Cert(t, pub, priv)
	signerCertPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	protected := []byte(`{"alg":"EdDSA"}`)

	protectedB64 := base64.RawURLEncoding.EncodeToString(protected)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	signingInput := []byte(protectedB64 + "." + payloadB64)
	sig := ed25519.Sign(priv, signingInput)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return string(signerCertPEMBytes), protectedB64 + "." + payloadB64 + "." + sigB64, payload
}

func newSelfSignedEd25519Cert(t *testing.T, pub ed25519.PublicKey, priv ed25519.PrivateKey) []byte {
	t.Helper()

	template := &stdlibx509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}

	der, err := stdlibx509.CreateCertificate(rand.Reader, template, template, pub, priv)
	require.NoError(t, err)

	return der
}

var _ validation.RuntimeMode = testRuntimeMode{}

func TestMachineConfigChuboModuleServicesFromYAML(t *testing.T) {
	t.Parallel()

	const cfg = `apiVersion: chubo.dev/v1alpha1
kind: MachineConfig
spec:
  trust:
    token: token
    ca:
      crt: dummy-ca-crt
      key: dummy-ca-key
  modules:
    chubo:
      enabled: true
      nomad:
        enabled: true
        role: server
      consul:
        enabled: true
        role: client
      openbao:
        enabled: true
        mode: nomadJob
`

	var mc MachineConfigV1Alpha1

	require.NoError(t, yaml.Unmarshal([]byte(cfg), &mc))
	require.NotNil(t, mc.Spec.Modules)
	require.NotNil(t, mc.Spec.Modules.Chubo)
	require.NotNil(t, mc.Spec.Modules.Chubo.Nomad)
	require.NotNil(t, mc.Spec.Modules.Chubo.Consul)
	require.NotNil(t, mc.Spec.Modules.Chubo.OpenBao)
	require.Equal(t, "server", mc.Spec.Modules.Chubo.Nomad.Role)
	require.Equal(t, "client", mc.Spec.Modules.Chubo.Consul.Role)
	require.Equal(t, "nomadJob", mc.Spec.Modules.Chubo.OpenBao.Mode)

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)
}

func TestMachineConfigNomadRendersOpenWontonFiles(t *testing.T) {
	t.Parallel()

	enabled := true

	mc := NewMachineConfigV1Alpha1()
	mc.Spec.Trust = &TrustSpec{
		Token: "token",
		CA: &CASpec{
			Crt: "dummy-ca-crt",
			Key: "dummy-ca-key",
		},
	}
	mc.Spec.Modules = &ModulesSpec{
		Chubo: &ChuboModuleSpec{
			Nomad: &ChuboRoleSpec{
				Enabled: &enabled,
				Role:    "server",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	var (
		foundConfig bool
		foundRole   bool
	)

	for _, f := range cfg.MachineConfig.MachineFiles {
		switch f.FilePath {
		case chuboOpenWontonConfigPath:
			foundConfig = true
			require.Contains(t, f.FileContent, `data_dir = "/var/lib/chubo/openwonton"`)
			require.Contains(t, f.FileContent, "server {\n  enabled = true")
		case chuboOpenWontonRolePath:
			foundRole = true
			require.Equal(t, "server\n", f.FileContent)
		}
	}

	require.True(t, foundConfig)
	require.True(t, foundRole)
}

func TestMachineConfigNomadInvalidRoleErrors(t *testing.T) {
	t.Parallel()

	mc := NewMachineConfigV1Alpha1()
	mc.Spec.Trust = &TrustSpec{
		Token: "token",
		CA: &CASpec{
			Crt: "dummy-ca-crt",
			Key: "dummy-ca-key",
		},
	}
	mc.Spec.Modules = &ModulesSpec{
		Chubo: &ChuboModuleSpec{
			Nomad: &ChuboRoleSpec{
				Role: "broken",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown spec.modules.chubo.nomad.role "broken"`)
}
