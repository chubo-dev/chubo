// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistOpenBaoInitCreatesParentDir(t *testing.T) {
	t.Parallel()

	orig := openBaoInitPath
	tmp := t.TempDir()
	openBaoInitPath = filepath.Join(tmp, "certs", "openbao-init.json")
	t.Cleanup(func() {
		openBaoInitPath = orig
	})

	resp := openBaoInitResponse{
		RootToken:  "root-token",
		KeysBase64: []string{"unseal-key"},
	}

	if err := persistOpenBaoInit(resp); err != nil {
		t.Fatalf("persistOpenBaoInit() error = %v", err)
	}

	if _, err := os.Stat(openBaoInitPath); err != nil {
		t.Fatalf("expected init file to exist: %v", err)
	}

	got, err := readOpenBaoInit()
	if err != nil {
		t.Fatalf("readOpenBaoInit() error = %v", err)
	}

	if got.RootToken != resp.RootToken {
		t.Fatalf("root token = %q, want %q", got.RootToken, resp.RootToken)
	}
	if len(got.KeysBase64) != 1 || got.KeysBase64[0] != resp.KeysBase64[0] {
		t.Fatalf("keys = %#v, want %#v", got.KeysBase64, resp.KeysBase64)
	}
}
