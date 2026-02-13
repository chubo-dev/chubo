package services

import (
	"context"
	stdlibx509 "crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"testing"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/cosi-project/runtime/pkg/state/impl/inmem"
	"github.com/cosi-project/runtime/pkg/state/impl/namespaced"
	"github.com/cosi-project/runtime/pkg/state/registry"
	siderox509 "github.com/siderolabs/crypto/x509"
	"github.com/stretchr/testify/require"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	talosconfig "github.com/chubo-dev/chubo/pkg/machinery/config"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
)

func TestEnsureChuboServiceTLSMaterialRotatesOnSANOrCAChange(t *testing.T) {
	t.Setenv("PLATFORM", "container")

	ctx := context.Background()

	resources := state.WrapCore(namespaced.NewState(
		func(ns resource.Namespace) state.CoreState {
			return inmem.NewState(ns)
		},
	))

	r := testRuntime{st: testState{v1a2: testV1Alpha2State{resources: resources}}}

	// Seed OS issuing CA + initial SANs.
	ca1, err := siderox509.NewSelfSignedCertificateAuthority(siderox509.Organization("chubo-test-ca1"))
	require.NoError(t, err)

	root := secrets.NewOSRoot(secrets.OSRootID)
	root.TypedSpec().IssuingCA = &siderox509.PEMEncodedCertificateAndKey{Crt: ca1.CrtPEM, Key: ca1.KeyPEM}
	require.NoError(t, resources.Create(ctx, root))

	sans := secrets.NewCertSAN(secrets.NamespaceName, secrets.CertSANAPIID)
	sans.TypedSpec().Append("node.test", "10.0.0.10")
	sans.TypedSpec().FQDN = "node.test"
	require.NoError(t, resources.Create(ctx, sans))

	// Redirect writes under a temp dir.
	oldRoot := chuboTLSRootDir
	chuboTLSRootDir = t.TempDir()
	t.Cleanup(func() { chuboTLSRootDir = oldRoot })

	// Initial material generation.
	require.NoError(t, EnsureChuboServiceTLSMaterial(ctx, r, OpenWontonServiceID))

	paths, ok := chuboServiceTLSPaths(OpenWontonServiceID)
	require.True(t, ok)

	cert1PEM, err := os.ReadFile(paths.Cert)
	require.NoError(t, err)

	cert1, err := readPEMCertificate(paths.Cert)
	require.NoError(t, err)
	require.True(t, certHasAllDNSNames(cert1, []string{"node.test"}))
	require.True(t, certHasAllIPs(cert1, []net.IP{netip.MustParseAddr("10.0.0.10").AsSlice()}))

	// No churn when up-to-date.
	require.NoError(t, EnsureChuboServiceTLSMaterial(ctx, r, OpenWontonServiceID))

	cert1PEMAfter, err := os.ReadFile(paths.Cert)
	require.NoError(t, err)
	require.Equal(t, cert1PEM, cert1PEMAfter)

	// Rotate when SAN changes.
	_, err = safe.StateUpdateWithConflicts(ctx, resources,
		secrets.NewCertSAN(secrets.NamespaceName, secrets.CertSANAPIID).Metadata(),
		func(res *secrets.CertSAN) error {
			res.TypedSpec().AppendIPs(netip.MustParseAddr("10.0.0.11"))
			return nil
		})
	require.NoError(t, err)

	require.NoError(t, EnsureChuboServiceTLSMaterial(ctx, r, OpenWontonServiceID))

	cert2, err := readPEMCertificate(paths.Cert)
	require.NoError(t, err)
	require.True(t, certHasAllIPs(cert2, []net.IP{netip.MustParseAddr("10.0.0.11").AsSlice()}))

	// Rotate when OS issuing CA changes.
	ca2, err := siderox509.NewSelfSignedCertificateAuthority(siderox509.Organization("chubo-test-ca2"))
	require.NoError(t, err)

	_, err = safe.StateUpdateWithConflicts(ctx, resources,
		secrets.NewOSRoot(secrets.OSRootID).Metadata(),
		func(res *secrets.OSRoot) error {
			res.TypedSpec().IssuingCA = &siderox509.PEMEncodedCertificateAndKey{Crt: ca2.CrtPEM, Key: ca2.KeyPEM}
			return nil
		})
	require.NoError(t, err)

	require.NoError(t, EnsureChuboServiceTLSMaterial(ctx, r, OpenWontonServiceID))

	caOnDisk, err := os.ReadFile(paths.CA)
	require.NoError(t, err)
	require.Equal(t, ca2.CrtPEM, caOnDisk)

	cert3, err := readPEMCertificate(paths.Cert)
	require.NoError(t, err)

	ca2Cert, err := func() (*stdlibx509.Certificate, error) {
		block, _ := pem.Decode(ca2.CrtPEM)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("failed to decode CA PEM")
		}
		return stdlibx509.ParseCertificate(block.Bytes)
	}()
	require.NoError(t, err)
	require.NoError(t, cert3.CheckSignatureFrom(ca2Cert))
}

type testRuntime struct {
	st runtime.State
}

func (r testRuntime) State() runtime.State {
	return r.st
}

type testState struct {
	v1a2 runtime.V1Alpha2State
}

func (s testState) Platform() runtime.Platform {
	return nil
}

func (s testState) Machine() runtime.MachineState {
	return nil
}

func (s testState) Cluster() runtime.ClusterState {
	return nil
}

func (s testState) V1Alpha2() runtime.V1Alpha2State {
	return s.v1a2
}

type testV1Alpha2State struct {
	resources state.State
}

func (s testV1Alpha2State) Resources() state.State {
	return s.resources
}

func (s testV1Alpha2State) NamespaceRegistry() *registry.NamespaceRegistry {
	return nil
}

func (s testV1Alpha2State) ResourceRegistry() *registry.ResourceRegistry {
	return nil
}

func (s testV1Alpha2State) GetConfig(context.Context) (talosconfig.Provider, error) {
	return nil, errors.New("not implemented")
}

func (s testV1Alpha2State) SetConfig(context.Context, string, talosconfig.Provider) error {
	return errors.New("not implemented")
}
