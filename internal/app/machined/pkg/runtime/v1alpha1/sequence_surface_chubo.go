// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo || chuboos

package v1alpha1

import "github.com/siderolabs/talos/internal/app/machined/pkg/runtime"

func initializeEarlyServicesTasks() []runtime.TaskSetupFunc {
	return []runtime.TaskSetupFunc{
		StartUdevd,
		StartMachined,
		StartAuditd,
		StartSyslogd,
		StartContainerd,
	}
}

func shouldStartDashboard(runtime.Mode) bool {
	// Chubo keeps dashboard disabled to reduce exposed process surface.
	return false
}
