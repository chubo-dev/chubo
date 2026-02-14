// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package v1alpha1

import "github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"

func shouldRunLeavePhase(_ runtime.Runtime) bool {
	// For Chubo, openwonton/opengyoza can run on non-controlplane nodes, so
	// always attempt the leave step during reset (the task itself is a no-op
	// when services are not configured).
	return true
}
