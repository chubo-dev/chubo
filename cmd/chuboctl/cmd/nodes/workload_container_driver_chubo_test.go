// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package nodes

import (
	"testing"

	"github.com/chubo-dev/chubo/pkg/machinery/api/common"
)

func TestWorkloadContainerDriverChubo(t *testing.T) {
	t.Parallel()

	if got := workloadContainerDriver(); got != common.ContainerDriver_CONTAINERD {
		t.Fatalf("expected CONTAINERD driver, got %v", got)
	}
}
