// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package cmd exposes the chuboctl entrypoint while cmd/talosctl is still in
// place as a compatibility namespace.
package cmd

import taloscmd "github.com/chubo-dev/chubo/cmd/talosctl/cmd"

// Execute runs the chuboctl root command.
func Execute() error {
	return taloscmd.Execute()
}
