// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"

	"github.com/chubo-dev/chubo/pkg/machinery/resources/k8s"
)

// NodeName implements the Runtime interface.
func (r *Runtime) NodeName() (string, error) {
	nodenameResource, err := safe.ReaderGet[*k8s.Nodename](
		context.Background(),
		r.s.V1Alpha2().Resources(),
		resource.NewMetadata(k8s.NamespaceName, k8s.NodenameType, k8s.NodenameID, resource.VersionUndefined),
	)
	if err != nil {
		return "", fmt.Errorf("error getting nodename resource: %w", err)
	}

	return nodenameResource.TypedSpec().Nodename, nil
}
