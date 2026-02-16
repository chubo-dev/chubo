// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/chubo-dev/chubo/pkg/machinery/api/common"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func TestGetContainerInspectorChuboRejectsUnsupportedDriver(t *testing.T) {
	t.Parallel()

	_, err := getContainerInspector(context.Background(), constants.WorkloadContainerdNamespace, common.ContainerDriver_CRI)
	if err == nil {
		t.Fatalf("expected an error for unsupported driver")
	}

	if !strings.Contains(err.Error(), "not available in chubo mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetContainerInspectorChuboRejectsUnsupportedNamespace(t *testing.T) {
	t.Parallel()

	_, err := getContainerInspector(context.Background(), "unknown", common.ContainerDriver_CONTAINERD)
	if err == nil {
		t.Fatalf("expected an error for unsupported namespace")
	}

	if !strings.Contains(err.Error(), "namespace") {
		t.Fatalf("unexpected error: %v", err)
	}
}
