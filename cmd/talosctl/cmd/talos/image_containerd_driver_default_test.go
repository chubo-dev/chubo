// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package talos

import (
	"testing"

	"github.com/chubo-dev/chubo/pkg/machinery/api/common"
)

func TestSystemContainerDriverDefault(t *testing.T) {
	t.Parallel()

	if got := systemContainerDriver(); got != common.ContainerDriver_CONTAINERD {
		t.Fatalf("expected containerd driver, got %v", got)
	}
}
