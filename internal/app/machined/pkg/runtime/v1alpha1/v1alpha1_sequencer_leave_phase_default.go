// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo && !chuboos

package v1alpha1

import (
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
)

func shouldRunLeavePhase(r runtime.Runtime) bool {
	return r.Config().Machine().Type() != machine.TypeWorker
}
