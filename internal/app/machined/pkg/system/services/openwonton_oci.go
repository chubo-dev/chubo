// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

type serviceReleaseOCITar struct {
	ServiceName string
	Version     string
	AssetURLs   map[string]string // key: runtime.GOARCH
}

var openWontonOCIRelease = serviceReleaseOCITar{
	ServiceName: "openwonton",
	Version:     "v1.6.5-rc1",
	AssetURLs: map[string]string{
		"amd64": "https://github.com/openwonton/openwonton/releases/download/v1.6.5-rc1/openwonton_release_linux_amd64_1.6.5-rc1_701cdab0b3bf3b834fbc6319ae08fddbebb42b80.docker.tar",
		"arm64": "https://github.com/openwonton/openwonton/releases/download/v1.6.5-rc1/openwonton_release_linux_arm64_1.6.5-rc1_701cdab0b3bf3b834fbc6319ae08fddbebb42b80.docker.tar",
	},
}

func ensureOpenWontonRuntime(ctx context.Context, targetBinaryPath, glibcDir string) error {
	// Fast path: already installed.
	if _, _, err := openWontonLoaderAndLibraryPath(glibcDir); err == nil {
		if st, err := os.Stat(targetBinaryPath); err == nil && st.Mode().IsRegular() && st.Mode()&0o111 != 0 {
			return nil
		}
	}

	arch := goruntime.GOARCH
	assetURL, ok := openWontonOCIRelease.AssetURLs[arch]
	if !ok {
		return fmt.Errorf("%s has no OCI release asset URL for arch %q", openWontonOCIRelease.ServiceName, arch)
	}

	cacheDir := filepath.Join(chuboArtifactsPath, openWontonOCIRelease.ServiceName, openWontonOCIRelease.Version, arch)
	archivePath := filepath.Join(cacheDir, filepath.Base(assetURL))
	layoutDir := filepath.Join(cacheDir, "oci")

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("failed to create openwonton cache dir %q: %w", cacheDir, err)
	}

	if _, err := os.Stat(archivePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat cached OCI tar %q: %w", archivePath, err)
		}

		if err := downloadReleaseArchive(ctx, assetURL, archivePath); err != nil {
			return err
		}
	}

	if _, err := os.Stat(filepath.Join(layoutDir, "index.json")); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat OCI layout dir %q: %w", layoutDir, err)
		}

		_ = os.RemoveAll(layoutDir)
		if err := extractTarToDir(archivePath, layoutDir); err != nil {
			return err
		}
	}

	layerPaths, err := ociLayoutLayerPaths(layoutDir, arch)
	if err != nil {
		return err
	}

	glibcTmp := glibcDir + ".part"
	_ = os.RemoveAll(glibcTmp)
	if err := os.MkdirAll(glibcTmp, 0o755); err != nil {
		return fmt.Errorf("failed to create openwonton glibc temp dir %q: %w", glibcTmp, err)
	}

	binTmp := targetBinaryPath + ".part"
	_ = os.Remove(binTmp)

	var extractedBinary bool

	for _, layerPath := range layerPaths {
		if err := extractOpenWontonLayer(layerPath, binTmp, glibcTmp, &extractedBinary); err != nil {
			return err
		}
	}

	if !extractedBinary {
		return errors.New("failed to locate bin/wonton in OCI image layers")
	}

	// Atomically replace the glibc directory.
	_ = os.RemoveAll(glibcDir)
	if err := os.Rename(glibcTmp, glibcDir); err != nil {
		return fmt.Errorf("failed to place openwonton glibc dir %q: %w", glibcDir, err)
	}

	// Atomically replace the binary.
	if err := os.Rename(binTmp, targetBinaryPath); err != nil {
		return fmt.Errorf("failed to place openwonton binary %q: %w", targetBinaryPath, err)
	}

	if err := os.Chmod(targetBinaryPath, 0o755); err != nil {
		return fmt.Errorf("failed to chmod openwonton binary %q: %w", targetBinaryPath, err)
	}

	// Validate the extracted runtime looks runnable.
	if _, _, err := openWontonLoaderAndLibraryPath(glibcDir); err != nil {
		return err
	}

	return nil
}

func openWontonLoaderAndLibraryPath(glibcDir string) (loaderPath, libraryPath string, err error) {
	var loaderBasename, triplet string

	switch goruntime.GOARCH {
	case "amd64":
		loaderBasename = "ld-linux-x86-64.so.2"
		triplet = "x86_64-linux-gnu"
	case "arm64":
		loaderBasename = "ld-linux-aarch64.so.1"
		triplet = "aarch64-linux-gnu"
	default:
		return "", "", fmt.Errorf("openwonton glibc loader unsupported arch %q", goruntime.GOARCH)
	}

	candidates := []string{
		filepath.Join(glibcDir, "lib64", loaderBasename),
		filepath.Join(glibcDir, "lib", loaderBasename),
		filepath.Join(glibcDir, "usr", "lib64", loaderBasename),
		filepath.Join(glibcDir, "usr", "lib", loaderBasename),
		filepath.Join(glibcDir, loaderBasename),
	}

	for _, p := range candidates {
		if st, statErr := os.Stat(p); statErr == nil && st.Mode().IsRegular() {
			loaderPath = p
			break
		}
	}

	if loaderPath == "" {
		return "", "", fmt.Errorf("openwonton glibc loader %q not found under %q", loaderBasename, glibcDir)
	}

	// Be liberal: include common locations. Non-existent paths are ignored by the loader.
	paths := []string{
		glibcDir,
		filepath.Join(glibcDir, "lib"),
		filepath.Join(glibcDir, "lib64"),
		filepath.Join(glibcDir, "usr", "lib"),
		filepath.Join(glibcDir, "usr", "lib64"),
		filepath.Join(glibcDir, "lib", triplet),
		filepath.Join(glibcDir, "usr", "lib", triplet),
	}

	return loaderPath, strings.Join(paths, ":"), nil
}

type ociIndex struct {
	Manifests []ociDescriptor `json:"manifests"`
}

type ociManifest struct {
	Layers []ociDescriptor `json:"layers"`
}

type ociDescriptor struct {
	Digest   string       `json:"digest"`
	Platform *ociPlatform `json:"platform,omitempty"`
}

type ociPlatform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

func ociLayoutLayerPaths(layoutDir, arch string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(layoutDir, "index.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read OCI index.json: %w", err)
	}

	var idx ociIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse OCI index.json: %w", err)
	}

	if len(idx.Manifests) == 0 {
		return nil, errors.New("OCI index.json has no manifests")
	}

	manifestDigest := idx.Manifests[0].Digest

	for _, m := range idx.Manifests {
		if m.Platform == nil {
			continue
		}

		if m.Platform.OS == "linux" && m.Platform.Architecture == arch {
			manifestDigest = m.Digest
			break
		}
	}

	manifestHex, err := digestHex(manifestDigest)
	if err != nil {
		return nil, fmt.Errorf("invalid OCI manifest digest %q: %w", manifestDigest, err)
	}

	manifestPath := filepath.Join(layoutDir, "blobs", "sha256", manifestHex)
	raw, err = os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read OCI manifest blob %q: %w", manifestPath, err)
	}

	var mf ociManifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, fmt.Errorf("failed to parse OCI manifest blob %q: %w", manifestPath, err)
	}

	if len(mf.Layers) == 0 {
		return nil, errors.New("OCI manifest has no layers")
	}

	out := make([]string, 0, len(mf.Layers))
	for _, layer := range mf.Layers {
		hex, err := digestHex(layer.Digest)
		if err != nil {
			return nil, fmt.Errorf("invalid OCI layer digest %q: %w", layer.Digest, err)
		}

		out = append(out, filepath.Join(layoutDir, "blobs", "sha256", hex))
	}

	return out, nil
}

func digestHex(digest string) (string, error) {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return "", errors.New("empty digest")
	}

	const prefix = "sha256:"
	if !strings.HasPrefix(digest, prefix) {
		return "", fmt.Errorf("unsupported digest %q", digest)
	}

	hex := strings.TrimPrefix(digest, prefix)
	if len(hex) != 64 {
		return "", fmt.Errorf("unexpected digest length %d", len(hex))
	}

	return hex, nil
}

func extractTarToDir(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open OCI tar %q: %w", archivePath, err)
	}
	defer f.Close() //nolint:errcheck

	tr := tar.NewReader(f)

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed reading OCI tar %q: %w", archivePath, err)
		}

		name := cleanTarPath(hdr.Name)
		if name == "" {
			continue
		}

		dst := filepath.Join(destDir, name)
		if !withinDir(destDir, dst) {
			return fmt.Errorf("refusing to extract path %q outside %q", hdr.Name, destDir)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return fmt.Errorf("failed to create OCI dir %q: %w", dst, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return fmt.Errorf("failed to create OCI parent dir for %q: %w", dst, err)
			}

			if err := writeFileAtomic(dst, tr, 0o644); err != nil {
				return err
			}
		default:
			// Ignore special entries in the layout tar.
		}
	}

	return nil
}

func extractOpenWontonLayer(layerPath, binTmp, glibcTmp string, extractedBinary *bool) error {
	layerFile, err := os.Open(layerPath)
	if err != nil {
		return fmt.Errorf("failed to open OCI layer %q: %w", layerPath, err)
	}
	defer layerFile.Close() //nolint:errcheck

	var r io.Reader = layerFile

	var header [2]byte
	n, _ := io.ReadFull(layerFile, header[:])
	if _, err := layerFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to rewind OCI layer %q: %w", layerPath, err)
	}

	if n == 2 && header[0] == 0x1f && header[1] == 0x8b {
		gz, err := gzip.NewReader(layerFile)
		if err != nil {
			return fmt.Errorf("failed to init gzip reader for OCI layer %q: %w", layerPath, err)
		}
		defer gz.Close() //nolint:errcheck

		r = gz
	}

	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed reading OCI layer tar %q: %w", layerPath, err)
		}

		name := cleanTarPath(hdr.Name)
		if name == "" {
			continue
		}

		// Extract the OpenWonton agent binary.
		if name == "bin/wonton" && (hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA) {
			if err := writeFileAtomic(binTmp, tr, 0o755); err != nil {
				return err
			}
			*extractedBinary = true
			continue
		}

		// Extract the glibc runtime bundle (shared libs + loader). We keep only common
		// lib locations to keep the footprint predictable.
		if !strings.HasPrefix(name, "lib/") &&
			!strings.HasPrefix(name, "lib64/") &&
			!strings.HasPrefix(name, "usr/lib/") &&
			!strings.HasPrefix(name, "usr/lib64/") {
			continue
		}

		dst := filepath.Join(glibcTmp, name)
		if !withinDir(glibcTmp, dst) {
			return fmt.Errorf("refusing to extract glibc path %q outside %q", name, glibcTmp)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return fmt.Errorf("failed to create glibc dir %q: %w", dst, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return fmt.Errorf("failed to create glibc parent dir for %q: %w", dst, err)
			}

			perm := os.FileMode(hdr.Mode) & 0o777
			if perm == 0 {
				perm = 0o644
			}

			if err := writeFileAtomic(dst, tr, perm); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return fmt.Errorf("failed to create glibc parent dir for %q: %w", dst, err)
			}

			if err := writeSymlinkInRoot(glibcTmp, name, hdr.Linkname); err != nil {
				return err
			}
		case tar.TypeLink:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return fmt.Errorf("failed to create glibc parent dir for %q: %w", dst, err)
			}

			if err := writeHardlinkInRoot(glibcTmp, name, hdr.Linkname); err != nil {
				return err
			}
		default:
			// Ignore special files.
		}
	}

	return nil
}

func writeFileAtomic(dst string, r io.Reader, perm os.FileMode) error {
	partial := dst + ".part"

	f, err := os.OpenFile(partial, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to create %q: %w", partial, err)
	}

	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(partial)

		return fmt.Errorf("failed to write %q: %w", partial, err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(partial)

		return fmt.Errorf("failed to finalize %q: %w", partial, err)
	}

	if err := os.Rename(partial, dst); err != nil {
		_ = os.Remove(partial)

		return fmt.Errorf("failed to place %q: %w", dst, err)
	}

	if err := os.Chmod(dst, perm); err != nil {
		return fmt.Errorf("failed to chmod %q: %w", dst, err)
	}

	return nil
}

func cleanTarPath(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	name = strings.TrimPrefix(name, "./")
	name = filepath.Clean(name)
	if name == "." || name == string(filepath.Separator) {
		return ""
	}

	// OCI layers/layout should never contain absolute paths.
	if filepath.IsAbs(name) {
		return ""
	}

	// Reject traversal.
	if strings.HasPrefix(name, ".."+string(filepath.Separator)) || name == ".." {
		return ""
	}

	return name
}

func withinDir(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func writeSymlinkInRoot(rootDir, name, linkname string) error {
	name = cleanTarPath(name)
	if name == "" {
		return nil
	}

	dst := filepath.Join(rootDir, name)
	if !withinDir(rootDir, dst) {
		return fmt.Errorf("refusing to write symlink %q outside %q", name, rootDir)
	}

	// Rewrite absolute links to a relative path inside rootDir.
	if filepath.IsAbs(linkname) {
		targetRel := filepath.Clean(strings.TrimPrefix(linkname, "/"))
		if targetRel == "." || targetRel == "" || strings.HasPrefix(targetRel, ".."+string(filepath.Separator)) || targetRel == ".." {
			return fmt.Errorf("refusing to write unsafe absolute symlink %q -> %q", name, linkname)
		}

		targetAbs := filepath.Join(rootDir, targetRel)
		if !withinDir(rootDir, targetAbs) {
			return fmt.Errorf("refusing to write symlink %q to target outside root %q", name, rootDir)
		}

		rel, err := filepath.Rel(filepath.Dir(dst), targetAbs)
		if err == nil && rel != "" {
			linkname = rel
		} else {
			linkname = targetRel
		}
	}

	_ = os.Remove(dst)

	if err := os.Symlink(linkname, dst); err != nil {
		return fmt.Errorf("failed to create symlink %q -> %q: %w", dst, linkname, err)
	}

	return nil
}

func writeHardlinkInRoot(rootDir, name, target string) error {
	name = cleanTarPath(name)
	if name == "" {
		return nil
	}

	target = cleanTarPath(target)
	if target == "" {
		return nil
	}

	dst := filepath.Join(rootDir, name)
	src := filepath.Join(rootDir, target)

	if !withinDir(rootDir, dst) || !withinDir(rootDir, src) {
		return fmt.Errorf("refusing to write hardlink %q -> %q outside %q", name, target, rootDir)
	}

	// Ignore missing sources (some tarballs rely on ordering/whiteouts).
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to stat hardlink source %q: %w", src, err)
	}

	_ = os.Remove(dst)

	if err := os.Link(src, dst); err != nil {
		return fmt.Errorf("failed to create hardlink %q -> %q: %w", dst, src, err)
	}

	return nil
}
