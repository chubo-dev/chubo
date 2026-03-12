// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"crypto/ed25519"
	"crypto/rand"
	stdlibx509 "crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/siderolabs/go-pointer"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	chuboacl "github.com/chubo-dev/chubo/pkg/chubo/acl"
	"github.com/chubo-dev/chubo/pkg/machinery/config/validation"
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
	bootstrapExpect := 3

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
				Enabled:         &enabled,
				Role:            "server",
				BootstrapExpect: &bootstrapExpect,
				Join:            []string{"10.0.0.10", "10.0.0.11"},
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
			require.Contains(t, f.FileContent, "acl {\n  enabled = true\n}")
			require.NotContains(t, f.FileContent, "\ntoken = ")
			require.Contains(t, f.FileContent, "server {\n  enabled = true")
			require.Contains(t, f.FileContent, "bootstrap_expect = 3")
			require.Contains(t, f.FileContent, "server_join {")
			require.Contains(t, f.FileContent, `retry_join = ["10.0.0.10","10.0.0.11"]`)
			require.NotContains(t, f.FileContent, `"driver.raw_exec.enable"`)
			require.NotContains(t, f.FileContent, "allow_privileged = true")
		case chuboOpenWontonRolePath:
			foundRole = true
			require.Equal(t, "server\n", f.FileContent)
		}
	}

	require.True(t, foundConfig)
	require.True(t, foundRole)
}

func TestMachineConfigNomadClientRendersOpenWontonServers(t *testing.T) {
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
				Enabled:          &enabled,
				Role:             "client",
				Join:             []string{"10.0.0.10", "10.0.0.11:4647"},
				NetworkInterface: "enp0s2",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	foundConfig := false

	for _, f := range cfg.MachineConfig.MachineFiles {
		if f.FilePath != chuboOpenWontonConfigPath {
			continue
		}

		foundConfig = true
		require.Contains(t, f.FileContent, "server {\n  enabled = false")
		require.NotContains(t, f.FileContent, "server_join {")
		require.Contains(t, f.FileContent, "client {\n  enabled = true")
		require.Contains(t, f.FileContent, `"driver.raw_exec.enable" = "1"`)
		require.Contains(t, f.FileContent, "volumes {\n      enabled = true")
		require.Contains(t, f.FileContent, "allow_privileged = true")
		require.Contains(t, f.FileContent, `servers = ["10.0.0.10:4647","10.0.0.11:4647"]`)
		require.Contains(t, f.FileContent, `http = "{{ GetInterfaceIP \"enp0s2\" }}"`)
		require.Contains(t, f.FileContent, `rpc = "{{ GetInterfaceIP \"enp0s2\" }}"`)
		require.Contains(t, f.FileContent, `serf = "{{ GetInterfaceIP \"enp0s2\" }}"`)
		require.Contains(t, f.FileContent, `network_interface = "enp0s2"`)
	}

	require.True(t, foundConfig)
}

func TestMachineConfigNomadServerClientRendersOpenWontonServerAndClient(t *testing.T) {
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
				Enabled:          &enabled,
				Role:             "server-client",
				Join:             []string{"10.0.0.10", "10.0.0.11:4647"},
				NetworkInterface: "enp0s2",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	foundConfig := false

	for _, f := range cfg.MachineConfig.MachineFiles {
		if f.FilePath != chuboOpenWontonConfigPath {
			continue
		}

		foundConfig = true
		require.Contains(t, f.FileContent, "server {\n  enabled = true")
		require.Contains(t, f.FileContent, "client {\n  enabled = true")
		require.Contains(t, f.FileContent, `"driver.raw_exec.enable" = "1"`)
		require.Contains(t, f.FileContent, "volumes {\n      enabled = true")
		require.Contains(t, f.FileContent, "allow_privileged = true")
		require.Contains(t, f.FileContent, `servers = ["10.0.0.10:4647","10.0.0.11:4647"]`)
		require.Contains(t, f.FileContent, `http = "{{ GetInterfaceIP \"enp0s2\" }}"`)
		require.Contains(t, f.FileContent, `rpc = "{{ GetInterfaceIP \"enp0s2\" }}"`)
		require.Contains(t, f.FileContent, `serf = "{{ GetInterfaceIP \"enp0s2\" }}"`)
		require.Contains(t, f.FileContent, `network_interface = "enp0s2"`)
	}

	require.True(t, foundConfig)
}

func TestMachineConfigNomadRendersConsulIntegrationWhenConsulEnabled(t *testing.T) {
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
				Enabled:          &enabled,
				Role:             "server-client",
				Join:             []string{"10.0.0.10", "10.0.0.11:4647"},
				NetworkInterface: "enp0s2",
			},
			Consul: &ChuboRoleSpec{
				Enabled: &enabled,
				Role:    "server-client",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	expectedConsulToken := chuboacl.WorkloadToken("token", "consul")
	foundConfig := false

	for _, f := range cfg.MachineConfig.MachineFiles {
		if f.FilePath != chuboOpenWontonConfigPath {
			continue
		}

		foundConfig = true
		require.Contains(t, f.FileContent, "consul {\n")
		require.Contains(t, f.FileContent, `address = "127.0.0.1:8500"`)
		require.Contains(t, f.FileContent, "ssl = true")
		require.Contains(t, f.FileContent, `token = "`+expectedConsulToken+`"`)
		require.Contains(t, f.FileContent, "verify_ssl = true")
		require.Contains(t, f.FileContent, `ca_file = "/var/lib/chubo/certs/opengyoza/ca.pem"`)
		require.Contains(t, f.FileContent, `cert_file = "/var/lib/chubo/certs/opengyoza/server.pem"`)
		require.Contains(t, f.FileContent, `key_file = "/var/lib/chubo/certs/opengyoza/server-key.pem"`)
		require.Contains(t, f.FileContent, `grpc_address = "127.0.0.1:8502"`)
		require.Contains(t, f.FileContent, `grpc_ca_file = "/var/lib/chubo/certs/opengyoza/ca.pem"`)
	}

	require.True(t, foundConfig)
}

func TestMachineConfigNomadRendersOpenBaoVaultIntegrationWhenOpenBaoExternalEnabled(t *testing.T) {
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
				Role:    "server-client",
			},
			OpenBao: &ChuboOpenBaoSpec{
				Enabled:                   &enabled,
				Mode:                      "external",
				VaultAddress:              "http://openbao.service.consul:8200",
				VaultToken:                "root-token",
				VaultAllowUnauthenticated: pointer.To(true),
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	foundConfig := false

	for _, f := range cfg.MachineConfig.MachineFiles {
		if f.FilePath != chuboOpenWontonConfigPath {
			continue
		}

		foundConfig = true
		require.Contains(t, f.FileContent, "vault {\n")
		require.Contains(t, f.FileContent, `enabled = true`)
		require.Contains(t, f.FileContent, `address = "http://openbao.service.consul:8200"`)
		require.Contains(t, f.FileContent, `allow_unauthenticated = true`)
		require.Contains(t, f.FileContent, `token = "root-token"`)
	}

	require.True(t, foundConfig)
}

func TestMachineConfigNomadClientOmitsOpenBaoVaultServerToken(t *testing.T) {
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
				Role:    "client",
			},
			OpenBao: &ChuboOpenBaoSpec{
				Enabled:      &enabled,
				Mode:         "external",
				VaultToken:   "root-token",
				VaultAddress: "http://127.0.0.1:8200",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	foundConfig := false

	for _, f := range cfg.MachineConfig.MachineFiles {
		if f.FilePath != chuboOpenWontonConfigPath {
			continue
		}

		foundConfig = true
		require.Contains(t, f.FileContent, "vault {\n")
		require.NotContains(t, f.FileContent, `token = "root-token"`)
	}

	require.True(t, foundConfig)
}

func TestMachineConfigNomadJobModeOmitsOpenBaoVaultBlock(t *testing.T) {
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
				Role:    "server-client",
			},
			OpenBao: &ChuboOpenBaoSpec{
				Enabled:                   &enabled,
				Mode:                      "nomadJob",
				VaultAddress:              "http://openbao.service.consul:8200",
				VaultToken:                "root-token",
				VaultAllowUnauthenticated: pointer.To(true),
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	for _, f := range cfg.MachineConfig.MachineFiles {
		if f.FilePath != chuboOpenWontonConfigPath {
			continue
		}

		require.NotContains(t, f.FileContent, "vault {\n")
	}
}

func TestMachineConfigOpenBaoExternalModeWritesModeFileOnly(t *testing.T) {
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
				Role:    "server-client",
			},
			OpenBao: &ChuboOpenBaoSpec{
				Enabled:      &enabled,
				Mode:         "external",
				VaultAddress: "http://openbao.service.consul:8200",
				VaultToken:   "root-token",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	var (
		foundVaultBlock bool
		foundModeFile   bool
	)

	for _, f := range cfg.MachineConfig.MachineFiles {
		switch f.FilePath {
		case chuboOpenWontonConfigPath:
			foundVaultBlock = true
			require.Contains(t, f.FileContent, "vault {\n")
			require.Contains(t, f.FileContent, `address = "http://openbao.service.consul:8200"`)
		case chuboOpenBaoModePath:
			foundModeFile = true
			require.Equal(t, "create", f.FileOp)
			require.Equal(t, "external\n", f.FileContent)
		}
	}

	require.True(t, foundVaultBlock)
	require.True(t, foundModeFile)
}

func TestMachineConfigOpenBaoHostServiceWritesModeAndConfig(t *testing.T) {
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
				Enabled:          &enabled,
				Role:             "server-client",
				Join:             []string{"10.0.0.11", "10.0.0.12"},
				NetworkInterface: "enp0s2",
			},
			OpenBao: &ChuboOpenBaoSpec{
				Enabled:      &enabled,
				Mode:         "hostService",
				VaultAddress: "http://127.0.0.1:8200",
				VaultToken:   "root-token",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	var (
		foundVaultBlock bool
		foundModeFile   bool
		foundConfigFile bool
		foundHostSpec   bool
		foundJobFile    bool
	)

	for _, f := range cfg.MachineConfig.MachineFiles {
		switch f.FilePath {
		case chuboOpenWontonConfigPath:
			foundVaultBlock = true
			require.Contains(t, f.FileContent, "vault {\n")
			require.Contains(t, f.FileContent, `address = "http://127.0.0.1:8200"`)
			require.Contains(t, f.FileContent, `token = "root-token"`)
		case chuboOpenBaoModePath:
			foundModeFile = true
			require.Equal(t, "create", f.FileOp)
			require.Equal(t, "hostService\n", f.FileContent)
		case chuboOpenBaoConfigPath:
			foundConfigFile = true
			require.Equal(t, "create", f.FileOp)
			require.Contains(t, f.FileContent, `storage "raft"`)
			require.Contains(t, f.FileContent, `/var/lib/chubo/openbao/data`)
			require.Contains(t, f.FileContent, `leader_api_addr = "http://10.0.0.11:8200"`)
			require.Contains(t, f.FileContent, `leader_api_addr = "http://10.0.0.12:8200"`)
			require.Contains(t, f.FileContent, `api_addr = "http://{{ GetInterfaceIP \"enp0s2\" }}:8200"`)
			require.Contains(t, f.FileContent, `cluster_addr = "http://{{ GetInterfaceIP \"enp0s2\" }}:8201"`)
			require.NotContains(t, f.FileContent, `node_id = "local"`)
		case chuboOpenBaoHostSpecPath:
			foundHostSpec = true
			require.Equal(t, "create", f.FileOp)
			require.Contains(t, f.FileContent, `"networkInterface": "enp0s2"`)
			require.Contains(t, f.FileContent, `"retryJoin": [`)
		case chuboOpenBaoJobPath:
			foundJobFile = true
		}
	}

	require.True(t, foundVaultBlock)
	require.True(t, foundModeFile)
	require.True(t, foundConfigFile)
	require.True(t, foundHostSpec)
	require.False(t, foundJobFile)
}

func TestMachineConfigOpenBaoDefaultModeUsesExternal(t *testing.T) {
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
				Role:    "server-client",
			},
			OpenBao: &ChuboOpenBaoSpec{
				Enabled:      &enabled,
				VaultAddress: "http://openbao.service.consul:8200",
			},
		},
	}

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	var (
		foundVaultBlock bool
		foundModeFile   bool
		foundJobFile    bool
	)

	for _, f := range cfg.MachineConfig.MachineFiles {
		switch f.FilePath {
		case chuboOpenWontonConfigPath:
			foundVaultBlock = true
			require.Contains(t, f.FileContent, "vault {\n")
			require.Contains(t, f.FileContent, `address = "http://openbao.service.consul:8200"`)
		case chuboOpenBaoModePath:
			foundModeFile = true
			require.Equal(t, "create", f.FileOp)
			require.Equal(t, "external\n", f.FileContent)
		case chuboOpenBaoJobPath:
			foundJobFile = true
		}
	}

	require.True(t, foundVaultBlock)
	require.True(t, foundModeFile)
	require.False(t, foundJobFile)
}

func TestMachineConfigOpenBaoHostServiceRequiresNomadNetworkInterface(t *testing.T) {
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
				Role:    "server-client",
			},
			OpenBao: &ChuboOpenBaoSpec{
				Enabled: &enabled,
				Mode:    "hostService",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "openbao.mode=hostService requires nomad networkInterface")
}

func TestMachineConfigNomadInvalidNetworkInterfaceErrors(t *testing.T) {
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
				NetworkInterface: "   ",
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec.modules.chubo.nomad.networkInterface must not be empty")
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

func TestMachineConfigNomadInvalidBootstrapExpectErrors(t *testing.T) {
	t.Parallel()

	bootstrapExpect := -1

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
				BootstrapExpect: &bootstrapExpect,
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec.modules.chubo.nomad.bootstrapExpect must be >= 0")
}

func TestMachineConfigNomadEmptyJoinErrors(t *testing.T) {
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
				Join: []string{"", "10.0.0.1"},
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "spec.modules.chubo.nomad.join must not contain empty entries")
}

func TestMachineConfigConsulRendersOpenGyozaFiles(t *testing.T) {
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
			Consul: &ChuboRoleSpec{
				Role: "client",
				Join: []string{"10.0.0.20", "10.0.0.21"},
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
		case chuboOpenGyozaConfigPath:
			foundConfig = true
			require.Contains(t, f.FileContent, `data_dir = "/var/lib/chubo/opengyoza"`)
			require.Contains(t, f.FileContent, "acl {\n  enabled = true")
			require.Contains(t, f.FileContent, `master = "`+chuboacl.WorkloadToken("token", "consul")+`"`)
			require.Contains(t, f.FileContent, `agent = "`+chuboacl.WorkloadToken("token", "consul")+`"`)
			require.Contains(t, f.FileContent, "server = false")
			require.Contains(t, f.FileContent, `retry_join = ["10.0.0.20","10.0.0.21"]`)
		case chuboOpenGyozaRolePath:
			foundRole = true
			require.Equal(t, "client\n", f.FileContent)
		}
	}

	require.True(t, foundConfig)
	require.True(t, foundRole)
}

func TestMachineConfigConsulServerClientRendersOpenGyozaServer(t *testing.T) {
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
			Consul: &ChuboRoleSpec{
				Role: "server-client",
				Join: []string{"10.0.0.20", "10.0.0.21"},
			},
		},
	}

	_, err := mc.Validate(testRuntimeMode{})
	require.NoError(t, err)

	cfg, err := mc.ToV1Alpha1()
	require.NoError(t, err)
	require.NotNil(t, cfg.MachineConfig)

	foundConfig := false

	for _, f := range cfg.MachineConfig.MachineFiles {
		if f.FilePath != chuboOpenGyozaConfigPath {
			continue
		}

		foundConfig = true
		require.Contains(t, f.FileContent, "server = true")
		require.Contains(t, f.FileContent, `retry_join = ["10.0.0.20","10.0.0.21"]`)
	}

	require.True(t, foundConfig)
}

func TestMachineConfigOpenBaoRendersNomadJobPayload(t *testing.T) {
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
			OpenBao: &ChuboOpenBaoSpec{
				Enabled: &enabled,
				Mode:    "nomadJob",
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
		foundMode    bool
	)

	for _, f := range cfg.MachineConfig.MachineFiles {
		switch f.FilePath {
		case chuboOpenBaoJobPath:
			foundPayload = true
			require.Contains(t, f.FileContent, `"ID": "openbao"`)
			require.Contains(t, f.FileContent, `"image": "ghcr.io/openbao/openbao:latest"`)
		case chuboOpenBaoModePath:
			foundMode = true
			require.Equal(t, "nomadJob\n", f.FileContent)
		}
	}

	require.True(t, foundPayload)
	require.True(t, foundMode)
}
