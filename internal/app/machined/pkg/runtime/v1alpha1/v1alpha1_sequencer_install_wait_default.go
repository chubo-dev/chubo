// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package v1alpha1

import (
	"context"

	"github.com/cosi-project/runtime/pkg/state"

	crires "github.com/chubo-dev/chubo/pkg/machinery/resources/cri"
)

func waitForInstallerImageCache(ctx context.Context, st state.State) error {
	return crires.WaitForImageCache(ctx, st)
}

func waitForInstallerImageCacheCopy(ctx context.Context, st state.State) error {
	return crires.WaitForImageCacheCopy(ctx, st)
}
