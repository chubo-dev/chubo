// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package talos

import "github.com/spf13/cobra"

func registerCgroupsResolveFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVar(target, "skip-cri-resolve", false, "do not resolve cgroup names via a request to CRI")
}
