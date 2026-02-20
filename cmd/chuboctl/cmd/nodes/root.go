// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package nodes re-exports node runtime commands from the legacy compatibility
// namespace while command paths are migrated.
package nodes

import (
	"github.com/spf13/cobra"

	legacytalos "github.com/chubo-dev/chubo/cmd/talosctl/cmd/talos"
)

// Commands is a list of commands published by the package.
var Commands []*cobra.Command

func init() {
	Commands = legacytalos.Commands
}
