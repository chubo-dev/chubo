// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func main() {
	if err := runDFShim(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runDFShim(args []string, output io.Writer) error {
	path, err := parseDFShimArgs(args)
	if err != nil {
		return err
	}

	mountPoint, err := resolveMountPoint(path)
	if err != nil {
		return err
	}

	var stats syscall.Statfs_t

	if err = syscall.Statfs(mountPoint, &stats); err != nil {
		return fmt.Errorf("df: statfs %s: %w", mountPoint, err)
	}

	blockBytes := uint64(stats.Bsize)
	totalKB := (stats.Blocks * blockBytes) / 1024
	usedKB := ((stats.Blocks - stats.Bfree) * blockBytes) / 1024
	availKB := (stats.Bavail * blockBytes) / 1024

	capacityPercent := 0
	if totalKB > 0 {
		capacityPercent = int((usedKB * 100) / totalKB)
	}

	_, err = fmt.Fprintf(output,
		"Filesystem 1024-blocks Used Available Capacity Mounted on\nchubo-df-shim %d %d %d %d%% %s\n",
		totalKB,
		usedKB,
		availKB,
		capacityPercent,
		mountPoint,
	)

	return err
}

func parseDFShimArgs(args []string) (string, error) {
	path := "/"

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "-k", "-P", "-kP", "-Pk":
				continue
			default:
				return "", fmt.Errorf("df: unsupported option %q", arg)
			}
		}

		path = arg
	}

	return path, nil
}

func resolveMountPoint(path string) (string, error) {
	cleanPath, err := normalizeDFPath(path)
	if err != nil {
		return "", err
	}

	if runtime.GOOS != "linux" {
		return cleanPath, nil
	}

	if mountPoint, loadErr := findMountPointFromFile(cleanPath, "/proc/self/mountinfo", mountPointFromMountInfo); loadErr == nil {
		return mountPoint, nil
	}

	if mountPoint, loadErr := findMountPointFromFile(cleanPath, "/proc/mounts", mountPointFromProcMounts); loadErr == nil {
		return mountPoint, nil
	}

	return "", fmt.Errorf("df: failed to determine mount point for %s", cleanPath)
}

func normalizeDFPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return "", fmt.Errorf("df: unable to resolve path %q: %w", path, err)
		}

		cleanPath = absPath
	}

	return cleanPath, nil
}

func findMountPointFromFile(path string, filePath string, finder func(string, string) string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	mountPoint := finder(path, string(data))
	if mountPoint == "" {
		return "", fmt.Errorf("mount point not found in %s", filePath)
	}

	return mountPoint, nil
}

func mountPointFromMountInfo(path string, raw string) string {
	best := ""

	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		mountPoint := unescapeMountField(fields[4])
		if !pathOnMount(path, mountPoint) {
			continue
		}

		if len(mountPoint) > len(best) {
			best = mountPoint
		}
	}

	return best
}

func mountPointFromProcMounts(path string, raw string) string {
	best := ""

	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		mountPoint := unescapeMountField(fields[1])
		if !pathOnMount(path, mountPoint) {
			continue
		}

		if len(mountPoint) > len(best) {
			best = mountPoint
		}
	}

	return best
}

func pathOnMount(path string, mountPoint string) bool {
	if mountPoint == "" {
		return false
	}

	if mountPoint == "/" {
		return true
	}

	if path == mountPoint {
		return true
	}

	return strings.HasPrefix(path, mountPoint+"/")
}

func unescapeMountField(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))

	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' || i+3 >= len(raw) {
			b.WriteByte(raw[i])
			continue
		}

		decoded, ok := decodeOctal(raw[i+1], raw[i+2], raw[i+3])
		if !ok {
			b.WriteByte(raw[i])
			continue
		}

		b.WriteByte(decoded)
		i += 3
	}

	return b.String()
}

func decodeOctal(a, b, c byte) (byte, bool) {
	if a < '0' || a > '7' || b < '0' || b > '7' || c < '0' || c > '7' {
		return 0, false
	}

	value := (a-'0')*64 + (b-'0')*8 + (c - '0')

	return value, true
}
