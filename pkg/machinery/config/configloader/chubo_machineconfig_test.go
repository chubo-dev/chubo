// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo || chuboos

package configloader_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chubo-dev/chubo/pkg/machinery/config/configloader"
)

type installMode struct{}

func (installMode) String() string      { return "install" }
func (installMode) RequiresInstall() bool { return true }
func (installMode) InContainer() bool   { return false }

func TestChuboOSMachineConfigDecodesAndSynthesizesV1Alpha1(t *testing.T) {
	t.Parallel()

	yaml := []byte(`
apiVersion: chubo.dev/v1alpha1
kind: MachineConfig
metadata:
  id: node-1
spec:
  install:
    disk: /dev/vdb
    wipe: false
    image: 10.0.2.2:5001/chubo/installer:dev
  trust:
    token: test-token
    ca:
      crt: |
        -----BEGIN CERTIFICATE-----
        TEST
        -----END CERTIFICATE-----
      key: |
        -----BEGIN PRIVATE KEY-----
        TEST
        -----END PRIVATE KEY-----
  registry:
    mirrors:
      "10.0.2.2:5001":
        endpoints:
          - "http://10.0.2.2:5001"
`)

	cfg, err := configloader.NewFromBytes(yaml)
	require.NoError(t, err)

	raw := cfg.RawV1Alpha1()
	require.NotNil(t, raw)
	require.NotNil(t, raw.MachineConfig)

	require.Equal(t, "controlplane", raw.MachineConfig.MachineType)
	require.Equal(t, "test-token", raw.MachineConfig.MachineToken)
	require.NotNil(t, raw.MachineConfig.MachineCA)
	require.NotEmpty(t, raw.MachineConfig.MachineCA.Crt)
	require.NotEmpty(t, raw.MachineConfig.MachineCA.Key)

	require.NotNil(t, raw.MachineConfig.MachineInstall)
	require.Equal(t, "/dev/vdb", raw.MachineConfig.MachineInstall.InstallDisk)
	require.Equal(t, "10.0.2.2:5001/chubo/installer:dev", raw.MachineConfig.MachineInstall.InstallImage)
	require.NotNil(t, raw.MachineConfig.MachineInstall.InstallWipe)
	require.False(t, *raw.MachineConfig.MachineInstall.InstallWipe)

	mirror, ok := raw.MachineConfig.MachineRegistries.RegistryMirrors["10.0.2.2:5001"]
	require.True(t, ok)
	require.Equal(t, []string{"http://10.0.2.2:5001"}, mirror.MirrorEndpoints)

	_, err = cfg.Validate(installMode{})
	require.NoError(t, err)
}

