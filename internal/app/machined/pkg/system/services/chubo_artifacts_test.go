// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureReleaseArchiveUsesBakedArchive(t *testing.T) {
	originalBakedPath := chuboBakedArtifactsPath
	chuboBakedArtifactsPath = filepath.Join(t.TempDir(), "baked")
	defer func() {
		chuboBakedArtifactsPath = originalBakedPath
	}()

	serviceName := "opengyoza"
	version := "v1.6.4"
	arch := "amd64"
	assetURL := "https://example.invalid/gyoza_1.6.4_linux_amd64.zip"
	bakedArchivePath := filepath.Join(chuboBakedArtifactsPath, serviceName, version, arch, filepath.Base(assetURL))
	cacheArchivePath := filepath.Join(t.TempDir(), "cache", filepath.Base(assetURL))
	want := []byte("baked-archive")

	if err := os.MkdirAll(filepath.Dir(bakedArchivePath), 0o755); err != nil {
		t.Fatalf("mkdir baked archive dir: %v", err)
	}

	if err := os.WriteFile(bakedArchivePath, want, 0o644); err != nil {
		t.Fatalf("write baked archive: %v", err)
	}

	if err := ensureReleaseArchive(context.Background(), serviceName, version, arch, assetURL, cacheArchivePath); err != nil {
		t.Fatalf("ensureReleaseArchive: %v", err)
	}

	got, err := os.ReadFile(cacheArchivePath)
	if err != nil {
		t.Fatalf("read cache archive: %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("cache archive mismatch: got %q want %q", got, want)
	}
}

func TestEnsureReleaseArchiveDownloadsWhenBakedArchiveMissing(t *testing.T) {
	originalBakedPath := chuboBakedArtifactsPath
	chuboBakedArtifactsPath = filepath.Join(t.TempDir(), "baked")
	defer func() {
		chuboBakedArtifactsPath = originalBakedPath
	}()

	want := []byte("downloaded-archive")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(want)
	}))
	defer server.Close()

	cacheArchivePath := filepath.Join(t.TempDir(), "cache", "archive.zip")

	if err := ensureReleaseArchive(context.Background(), "opengyoza", "v1.6.4", "amd64", server.URL+"/archive.zip", cacheArchivePath); err != nil {
		t.Fatalf("ensureReleaseArchive: %v", err)
	}

	got, err := os.ReadFile(cacheArchivePath)
	if err != nil {
		t.Fatalf("read cache archive: %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("cache archive mismatch: got %q want %q", got, want)
	}
}
