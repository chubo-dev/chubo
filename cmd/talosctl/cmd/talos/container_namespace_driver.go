// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package talos

import (
	"github.com/chubo-dev/chubo/pkg/machinery/api/common"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func namespaceAndDriverForFlag(workloadNamespace bool) (string, common.ContainerDriver) {
	if workloadNamespace {
		return constants.WorkloadContainerdNamespace, workloadContainerDriver()
	}

	return constants.SystemContainerdNamespace, common.ContainerDriver_CONTAINERD
}
