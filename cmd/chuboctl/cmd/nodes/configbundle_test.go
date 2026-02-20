// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package nodes //nolint:testpackage // test unexported bundle helpers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteConfigBundleDefaultSubdir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	bundle := mustTarGz(t, map[string]string{
		"nomad.env": "NOMAD_ADDR=http://127.0.0.1:4646\n",
		"ca.pem":    "-----BEGIN CERTIFICATE-----\n",
	})

	err := writeConfigBundle(root, io.NopCloser(bytes.NewReader(bundle)), "nomadconfig", false)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(root, "nomadconfig", "nomad.env"))
	require.NoError(t, err)
	require.Contains(t, string(data), "NOMAD_ADDR=")
}

func TestWriteConfigBundleExistingTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "nomadconfig")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "old"), []byte("old"), 0o600))

	bundle := mustTarGz(t, map[string]string{"nomad.env": "new\n"})

	err := writeConfigBundle(root, io.NopCloser(bytes.NewReader(bundle)), "nomadconfig", false)
	require.Error(t, err)

	err = writeConfigBundle(root, io.NopCloser(bytes.NewReader(bundle)), "nomadconfig", true)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(target, "old"))
	require.True(t, os.IsNotExist(err))

	data, err := os.ReadFile(filepath.Join(target, "nomad.env"))
	require.NoError(t, err)
	require.Equal(t, "new\n", string(data))
}

func TestWriteConfigBundleNonExistingPathIsTargetDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "custom-output")
	bundle := mustTarGz(t, map[string]string{"consul.env": "CONSUL_HTTP_ADDR=http://127.0.0.1:8500\n"})

	err := writeConfigBundle(target, io.NopCloser(bytes.NewReader(bundle)), "consulconfig", false)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(target, "consul.env"))
	require.NoError(t, err)
}

func TestWriteConfigBundleRejectsFilePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	filePath := filepath.Join(root, "output")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o600))

	bundle := mustTarGz(t, map[string]string{"openbao.env": "VAULT_ADDR=http://127.0.0.1:8200\n"})
	err := writeConfigBundle(filePath, io.NopCloser(bytes.NewReader(bundle)), "openbaoconfig", false)
	require.Error(t, err)
}

func mustTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o600,
			Size: int64(len(content)),
		}))

		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	return buf.Bytes()
}
