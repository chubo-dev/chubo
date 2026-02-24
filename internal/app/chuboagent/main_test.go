// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDFShim(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	if err := runDFShim([]string{"-kP", "/"}, &out); err != nil {
		t.Fatalf("runDFShim returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %q", len(lines), out.String())
	}

	if lines[0] != "Filesystem 1024-blocks Used Available Capacity Mounted on" {
		t.Fatalf("unexpected header line: %q", lines[0])
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 6 {
		t.Fatalf("expected at least 6 fields in body line, got %d: %q", len(fields), lines[1])
	}

	if fields[0] != "chubo-df-shim" {
		t.Fatalf("unexpected filesystem label: %q", fields[0])
	}
}

func TestParseDFShimArgsRejectsUnknownOption(t *testing.T) {
	t.Parallel()

	if _, err := parseDFShimArgs([]string{"--bogus"}); err == nil {
		t.Fatal("expected parseDFShimArgs to fail for unsupported option")
	}
}

func TestMountPointFromMountInfo(t *testing.T) {
	t.Parallel()

	raw := strings.Join([]string{
		"39 38 0:30 / /run rw,nosuid,nodev - tmpfs none rw",
		"54 38 253:4 / /var rw,nosuid,nodev - xfs /dev/vda4 rw",
	}, "\n")

	mountPoint := mountPointFromMountInfo("/var/lib/chubo/openwonton/alloc", raw)
	if mountPoint != "/var" {
		t.Fatalf("expected /var mount point, got %q", mountPoint)
	}
}

func TestMountPointFromProcMountsEscapes(t *testing.T) {
	t.Parallel()

	raw := strings.Join([]string{
		"none / tmpfs rw 0 0",
		"none /Volumes/My\\040Disk ext4 rw 0 0",
	}, "\n")

	mountPoint := mountPointFromProcMounts("/Volumes/My Disk/work", raw)
	if mountPoint != "/Volumes/My Disk" {
		t.Fatalf("expected escaped mount point to decode, got %q", mountPoint)
	}
}

func TestPathOnMount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		path       string
		mountPoint string
		expected   bool
	}{
		{path: "/var/lib/chubo", mountPoint: "/var", expected: true},
		{path: "/var/lib/chubo", mountPoint: "/", expected: true},
		{path: "/var/lib/chubo", mountPoint: "/var/lib/chubo", expected: true},
		{path: "/var/lib/chubo", mountPoint: "/tmp", expected: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.path+"-"+tc.mountPoint, func(t *testing.T) {
			t.Parallel()

			if got := pathOnMount(tc.path, tc.mountPoint); got != tc.expected {
				t.Fatalf("pathOnMount(%q, %q) = %v, expected %v", tc.path, tc.mountPoint, got, tc.expected)
			}
		})
	}
}

func TestNormalizeDFPath(t *testing.T) {
	t.Parallel()

	got, err := normalizeDFPath(".")
	if err != nil {
		t.Fatalf("normalizeDFPath returned error: %v", err)
	}

	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
}
