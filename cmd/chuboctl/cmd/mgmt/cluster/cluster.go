// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package cluster re-exports the local cluster command from the legacy
// compatibility namespace while command paths are migrated.
package cluster

import (
	"github.com/spf13/cobra"

	legacycluster "github.com/chubo-dev/chubo/cmd/talosctl/cmd/mgmt/cluster"
)

// Cmd represents the cluster command.
var Cmd *cobra.Command = legacycluster.Cmd
