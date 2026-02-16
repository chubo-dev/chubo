// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package talos

import "github.com/spf13/cobra"

func registerNetstatWorkloadFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVarP(target, "workloads", "P", false, "show sockets used by workload tasks")
	cmd.Flags().BoolVar(target, "pods", false, "alias for --workloads")
	cmd.Flags().MarkHidden("pods") //nolint:errcheck
}
