// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package v1alpha1

import (
	"context"

	"github.com/cosi-project/runtime/pkg/state"
)

// Chubo keeps installer pulls driven by registries config only and does not
// orchestrate image-cache resources.
func waitForInstallerImageCache(context.Context, state.State) error {
	return nil
}

// TODO(chubo): remove this shim once installer pulls no longer depend on
// CRI-based registry plumbing in the install path.
func waitForInstallerImageCacheCopy(context.Context, state.State) error {
	return nil
}
