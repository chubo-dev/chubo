// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package v1alpha1

import (
	"strconv"

	"github.com/siderolabs/go-pointer"
	"github.com/siderolabs/go-procfs/procfs"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func initializeEarlyServicesTasks() []runtime.TaskSetupFunc {
	return []runtime.TaskSetupFunc{
		StartUdevd,
		StartMachined,
		StartAuditd,
		StartSyslogd,
		StartContainerd,
	}
}

func shouldStartDashboard(mode runtime.Mode) bool {
	if mode == runtime.ModeMetalAgent {
		return false
	}

	disabledStr := procfs.ProcCmdline().Get(constants.KernelParamDashboardDisabled).First()
	disabled, _ := strconv.ParseBool(pointer.SafeDeref(disabledStr)) //nolint:errcheck

	return !disabled
}
