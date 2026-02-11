// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo && !chuboos

package runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/siderolabs/talos/internal/pkg/containers"
	taloscontainerd "github.com/siderolabs/talos/internal/pkg/containers/containerd"
	"github.com/siderolabs/talos/internal/pkg/containers/cri"
	"github.com/siderolabs/talos/pkg/machinery/api/common"
	"github.com/siderolabs/talos/pkg/machinery/constants"
)

func getContainerInspector(ctx context.Context, namespace string, driver common.ContainerDriver) (containers.Inspector, error) {
	switch driver {
	case common.ContainerDriver_CRI:
		if namespace != constants.K8sContainerdNamespace {
			return nil, errors.New("CRI inspector is supported only for K8s namespace")
		}

		return cri.NewInspector(ctx)
	case common.ContainerDriver_CONTAINERD:
		addr := constants.CRIContainerdAddress
		if namespace == constants.SystemContainerdNamespace {
			addr = constants.SystemContainerdAddress
		}

		return taloscontainerd.NewInspector(ctx, namespace, taloscontainerd.WithContainerdAddress(addr))
	default:
		return nil, fmt.Errorf("unsupported driver %q", driver)
	}
}
