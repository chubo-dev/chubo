// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo || chuboos

package container

import (
	chubotypes "github.com/siderolabs/talos/pkg/machinery/config/types/chuboos"
)

func maybeSynthesizeChuboOSV1Alpha1(container *Container) error {
	if container.v1alpha1Config != nil {
		// If the user supplied v1alpha1 directly, don't second-guess it.
		return nil
	}

	for _, doc := range container.documents {
		mc, ok := doc.(*chubotypes.MachineConfigV1Alpha1)
		if !ok {
			continue
		}

		v1, err := mc.ToV1Alpha1()
		if err != nil {
			return err
		}

		container.v1alpha1Config = v1

		return nil
	}

	return nil
}
