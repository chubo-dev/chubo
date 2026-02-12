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

	"github.com/chubo-dev/chubo/cmd/talosctl/pkg/talos/helpers"
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

func downloadBundle(ctx context.Context, raw rawDownloadFunc) (io.ReadCloser, error) {
	r, err := raw(ctx)
	if err != nil {
		return nil, fmt.Errorf("error downloading config bundle: %w", err)
	}

	return r, nil
}

func writeConfigBundle(localPath string, bundle io.ReadCloser, defaultDir string, force bool) error {
	if localPath == stdoutOutput {
		defer bundle.Close() //nolint:errcheck

		_, err := io.Copy(os.Stdout, bundle)
		return err
	}

	st, err := os.Stat(localPath)
	switch {
	case err == nil:
		if !st.IsDir() {
			return fmt.Errorf("local path %q should be a directory path for bundle output", localPath)
		}

		localPath = filepath.Join(localPath, defaultDir)
	case errors.Is(err, fs.ErrNotExist):
		// Treat non-existing path as the target bundle directory.
	default:
		return fmt.Errorf("error checking path %q: %w", localPath, err)
	}

	if _, err := os.Stat(localPath); err == nil && !force {
		return fmt.Errorf("%s already exists, use --force to overwrite: %q", defaultDir, localPath)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("error checking path %q: %w", localPath, err)
	}

	if force {
		if err := os.RemoveAll(localPath); err != nil {
			return fmt.Errorf("error removing existing path %q: %w", localPath, err)
		}
	}

	if err := os.MkdirAll(localPath, 0o755); err != nil {
		return fmt.Errorf("error creating bundle output directory %q: %w", localPath, err)
	}

	return helpers.ExtractTarGz(localPath, bundle)
}
