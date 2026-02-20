// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package mgmt re-exports the management command set from the legacy
// compatibility namespace while the migration to cmd/chuboctl is in progress.
package mgmt

import (
	"github.com/spf13/cobra"

	legacymgmt "github.com/chubo-dev/chubo/cmd/talosctl/cmd/mgmt"
)

// Commands is a list of commands published by the package.
var Commands []*cobra.Command

// GenV1Alpha1Config is kept with this historical name for backward compatibility.
var GenV1Alpha1Config = legacymgmt.GenV1Alpha1Config

func init() {
	Commands = legacymgmt.Commands
}
