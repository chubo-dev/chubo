// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package runtime

import (
	"context"
	"fmt"

	"github.com/chubo-dev/chubo/internal/pkg/containers"
	taloscontainerd "github.com/chubo-dev/chubo/internal/pkg/containers/containerd"
	"github.com/chubo-dev/chubo/pkg/machinery/api/common"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func getContainerInspector(ctx context.Context, namespace string, driver common.ContainerDriver) (containers.Inspector, error) {
	if driver != common.ContainerDriver_CONTAINERD {
		return nil, fmt.Errorf("driver %q is not available in chubo mode", driver)
	}

	addr := constants.CRIContainerdAddress
	if namespace == constants.SystemContainerdNamespace {
		addr = constants.SystemContainerdAddress
	}

	return taloscontainerd.NewInspector(ctx, namespace, taloscontainerd.WithContainerdAddress(addr))
}
