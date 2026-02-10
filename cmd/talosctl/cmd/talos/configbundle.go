// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package talos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/siderolabs/talos/cmd/talosctl/pkg/talos/helpers"
)

type rawDownloadFunc func(context.Context) (io.ReadCloser, error)

func defaultPath(args []string) (string, error) {
	if len(args) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("error getting current working directory: %w", err)
		}

		return wd, nil
	}

	return filepath.Clean(args[0]), nil
}

func downloadSingleFile(ctx context.Context, raw rawDownloadFunc, filename string) ([]byte, error) {
	r, err := raw(ctx)
	if err != nil {
		return nil, fmt.Errorf("error copying: %w", err)
	}

	return helpers.ExtractFileFromTarGz(filename, r)
}

func writeConfigFile(localPath string, data []byte, defaultFilename string, force bool) error {
	if localPath == stdoutOutput {
		_, err := os.Stdout.Write(data)

		return err
	}

	st, err := os.Stat(localPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("error checking path %q: %w", localPath, err)
		}

		// If path doesn't exist, treat it as a file path (mkdir parent).
		if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
			return err
		}
	} else if st.IsDir() {
		localPath = filepath.Join(localPath, defaultFilename)
	}

	if _, err := os.Stat(localPath); err == nil && !force {
		return fmt.Errorf("%s already exists, use --force to overwrite: %q", defaultFilename, localPath)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("error checking path %q: %w", localPath, err)
	}

	return os.WriteFile(localPath, data, 0o600)
}
