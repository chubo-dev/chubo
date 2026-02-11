// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

const (
	binaryModeArtifact = "artifact"
	binaryModeFallback = "fallback"
	binaryModeMissing  = "missing"
	binaryModeUnknown  = "unknown"
)

func detectServiceBinaryMode(targetPath string, fallbackPath string) string {
	targetHash, err := hashFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return binaryModeMissing
		}

		return binaryModeUnknown
	}

	fallbackHash, err := hashFile(fallbackPath)
	if err != nil {
		return binaryModeUnknown
	}

	if targetHash == fallbackHash {
		return binaryModeFallback
	}

	return binaryModeArtifact
}

func hashFile(path string) ([32]byte, error) {
	var zero [32]byte

	f, err := os.Open(path)
	if err != nil {
		return zero, err
	}

	defer f.Close() //nolint:errcheck

	h := sha256.New()

	if _, err = io.Copy(h, f); err != nil {
		return zero, fmt.Errorf("failed to hash file %q: %w", path, err)
	}

	sum := h.Sum(nil)

	return [32]byte(sum), nil
}
