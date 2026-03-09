// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"
)

const (
	chuboArtifactsPath     = "/var/lib/chubo/artifacts"
	releaseDownloadTimeout = 10 * time.Minute
)

var chuboBakedArtifactsPath = "/usr/local/lib/chubo-artifacts"

type serviceReleaseBinary struct {
	ServiceName string
	Version     string
	ZipEntry    string
	AssetURLs   map[string]string // key: runtime.GOARCH
}

type serviceReleaseTarGzBinary struct {
	ServiceName string
	Version     string
	TarEntry    string
	AssetURLs   map[string]string // key: runtime.GOARCH
}

func ensureServiceBinaryWithRelease(ctx context.Context, targetPath string, fallbackPath string, release serviceReleaseBinary) error {
	if err := installServiceBinaryFromRelease(ctx, targetPath, release); err == nil {
		return nil
	}

	return ensureServiceBinary(targetPath, fallbackPath)
}

func ensureServiceBinaryWithTarGzRelease(ctx context.Context, targetPath string, fallbackPath string, release serviceReleaseTarGzBinary) error {
	if err := installServiceBinaryFromTarGzRelease(ctx, targetPath, release); err == nil {
		return nil
	}

	return ensureServiceBinary(targetPath, fallbackPath)
}

func ensureServiceBinary(targetPath string, fallbackPath string) error {
	if st, err := os.Stat(targetPath); err == nil {
		if st.Mode()&0o111 == 0 {
			return os.Chmod(targetPath, 0o755)
		}

		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	src, err := os.Open(fallbackPath)
	if err != nil {
		return fmt.Errorf("failed to open fallback binary %q: %w", fallbackPath, err)
	}

	defer src.Close() //nolint:errcheck

	dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create service binary %q: %w", targetPath, err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()

		return fmt.Errorf("failed to copy fallback binary to %q: %w", targetPath, err)
	}

	if err := dst.Close(); err != nil {
		return fmt.Errorf("failed to finalize service binary %q: %w", targetPath, err)
	}

	return nil
}

func installServiceBinaryFromRelease(ctx context.Context, targetPath string, release serviceReleaseBinary) error {
	artifactBinaryPath, err := ensureCachedReleaseBinary(ctx, release)
	if err != nil {
		return err
	}

	return copyExecutable(artifactBinaryPath, targetPath)
}

func installServiceBinaryFromTarGzRelease(ctx context.Context, targetPath string, release serviceReleaseTarGzBinary) error {
	artifactBinaryPath, err := ensureCachedTarGzReleaseBinary(ctx, release)
	if err != nil {
		return err
	}

	return copyExecutable(artifactBinaryPath, targetPath)
}

func ensureCachedReleaseBinary(ctx context.Context, release serviceReleaseBinary) (string, error) {
	arch := goruntime.GOARCH

	assetURL, ok := release.AssetURLs[arch]
	if !ok {
		return "", fmt.Errorf("%s has no release asset URL for arch %q", release.ServiceName, arch)
	}

	cacheDir := filepath.Join(chuboArtifactsPath, release.ServiceName, release.Version, arch)
	archivePath := filepath.Join(cacheDir, filepath.Base(assetURL))
	extractedBinaryPath := filepath.Join(cacheDir, release.ZipEntry)

	if st, err := os.Stat(extractedBinaryPath); err == nil && st.Mode().IsRegular() {
		if st.Mode()&0o111 == 0 {
			if chmodErr := os.Chmod(extractedBinaryPath, 0o755); chmodErr != nil {
				return "", fmt.Errorf("failed to chmod cached binary %q: %w", extractedBinaryPath, chmodErr)
			}
		}

		return extractedBinaryPath, nil
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create artifact cache dir %q: %w", cacheDir, err)
	}

	if err := ensureReleaseArchive(ctx, release.ServiceName, release.Version, arch, assetURL, archivePath); err != nil {
		return "", err
	}

	if err := extractZipEntryToFile(archivePath, release.ZipEntry, extractedBinaryPath); err != nil {
		return "", err
	}

	return extractedBinaryPath, nil
}

func ensureCachedTarGzReleaseBinary(ctx context.Context, release serviceReleaseTarGzBinary) (string, error) {
	arch := goruntime.GOARCH

	assetURL, ok := release.AssetURLs[arch]
	if !ok {
		return "", fmt.Errorf("%s has no release asset URL for arch %q", release.ServiceName, arch)
	}

	cacheDir := filepath.Join(chuboArtifactsPath, release.ServiceName, release.Version, arch)
	archivePath := filepath.Join(cacheDir, filepath.Base(assetURL))
	extractedBinaryPath := filepath.Join(cacheDir, filepath.Base(release.TarEntry))

	if st, err := os.Stat(extractedBinaryPath); err == nil && st.Mode().IsRegular() {
		if st.Mode()&0o111 == 0 {
			if chmodErr := os.Chmod(extractedBinaryPath, 0o755); chmodErr != nil {
				return "", fmt.Errorf("failed to chmod cached binary %q: %w", extractedBinaryPath, chmodErr)
			}
		}

		return extractedBinaryPath, nil
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create artifact cache dir %q: %w", cacheDir, err)
	}

	if err := ensureReleaseArchive(ctx, release.ServiceName, release.Version, arch, assetURL, archivePath); err != nil {
		return "", err
	}

	if err := extractTarGzEntryToFile(archivePath, release.TarEntry, extractedBinaryPath); err != nil {
		return "", err
	}

	return extractedBinaryPath, nil
}

func downloadReleaseArchive(ctx context.Context, url string, archivePath string) error {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, releaseDownloadTimeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create release download request for %q: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download release archive from %q: %w", url, err)
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("release archive download failed for %q: status=%d", url, resp.StatusCode)
	}

	partialPath := archivePath + ".part"

	out, err := os.OpenFile(partialPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create partial archive %q: %w", partialPath, err)
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()

		return fmt.Errorf("failed to write release archive %q: %w", partialPath, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to finalize release archive %q: %w", partialPath, err)
	}

	if err := os.Rename(partialPath, archivePath); err != nil {
		return fmt.Errorf("failed to atomically place release archive %q: %w", archivePath, err)
	}

	return nil
}

func ensureReleaseArchive(ctx context.Context, serviceName, version, arch, assetURL, archivePath string) error {
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return fmt.Errorf("failed to create archive parent dir for %q: %w", archivePath, err)
	}

	if _, err := os.Stat(archivePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat cached archive %q: %w", archivePath, err)
	}

	if copied, err := copyBakedReleaseArchiveIfPresent(serviceName, version, arch, assetURL, archivePath); err != nil {
		return err
	} else if copied {
		return nil
	}

	return downloadReleaseArchive(ctx, assetURL, archivePath)
}

func copyBakedReleaseArchiveIfPresent(serviceName, version, arch, assetURL, archivePath string) (bool, error) {
	bakedPath := filepath.Join(chuboBakedArtifactsPath, serviceName, version, arch, filepath.Base(assetURL))

	st, err := os.Stat(bakedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to stat baked archive %q: %w", bakedPath, err)
	}

	if !st.Mode().IsRegular() {
		return false, fmt.Errorf("baked archive %q is not a regular file", bakedPath)
	}

	if err := copyFile(bakedPath, archivePath, 0o644); err != nil {
		return false, fmt.Errorf("failed to seed cached archive %q from baked archive %q: %w", archivePath, bakedPath, err)
	}

	return true, nil
}

func extractZipEntryToFile(archivePath string, zipEntry string, outputPath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open release archive %q: %w", archivePath, err)
	}

	defer reader.Close() //nolint:errcheck

	entry, err := findZipEntry(reader.File, zipEntry)
	if err != nil {
		return err
	}

	src, err := entry.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip entry %q in %q: %w", zipEntry, archivePath, err)
	}

	defer src.Close() //nolint:errcheck

	partialPath := outputPath + ".part"

	dst, err := os.OpenFile(partialPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create extracted binary %q: %w", partialPath, err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()

		return fmt.Errorf("failed to extract zip entry %q from %q: %w", zipEntry, archivePath, err)
	}

	if err := dst.Close(); err != nil {
		return fmt.Errorf("failed to finalize extracted binary %q: %w", partialPath, err)
	}

	if err := os.Rename(partialPath, outputPath); err != nil {
		return fmt.Errorf("failed to atomically place extracted binary %q: %w", outputPath, err)
	}

	return nil
}

func extractTarGzEntryToFile(archivePath, tarEntry, outputPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open release archive %q: %w", archivePath, err)
	}

	defer f.Close() //nolint:errcheck

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to open gzip stream for %q: %w", archivePath, err)
	}

	defer gr.Close() //nolint:errcheck

	tr := tar.NewReader(gr)
	expectedEntry := cleanTarPath(tarEntry)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("failed reading tar stream from %q: %w", archivePath, err)
		}

		name := cleanTarPath(hdr.Name)
		if name != expectedEntry {
			continue
		}

		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			return fmt.Errorf("tar entry %q in %q is not a regular file", tarEntry, archivePath)
		}

		perm := os.FileMode(hdr.Mode) & 0o777
		if perm&0o111 == 0 {
			perm = 0o755
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return fmt.Errorf("failed to create extracted binary parent dir %q: %w", filepath.Dir(outputPath), err)
		}

		return writeFileAtomicFromReader(outputPath, tr, perm)
	}

	return fmt.Errorf("tar entry %q not found", tarEntry)
}

func findZipEntry(files []*zip.File, expectedName string) (*zip.File, error) {
	for _, f := range files {
		if f.Name == expectedName {
			return f, nil
		}
	}

	for _, f := range files {
		if filepath.Base(strings.TrimSpace(f.Name)) == expectedName {
			return f, nil
		}
	}

	return nil, fmt.Errorf("zip entry %q not found", expectedName)
}

func copyExecutable(srcPath string, dstPath string) error {
	return copyFile(srcPath, dstPath, 0o755)
}

func copyFile(srcPath string, dstPath string, perm os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open file source %q: %w", srcPath, err)
	}

	defer src.Close() //nolint:errcheck

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("failed to create file target dir for %q: %w", dstPath, err)
	}

	return writeFileAtomicFromReader(dstPath, src, perm)
}
