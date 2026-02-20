// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Command talosctl is a legacy compatibility entrypoint for chuboctl.
package main

import (
	"os"

	_ "github.com/chubo-dev/chubo/cmd/talosctl/acompat"
	"github.com/chubo-dev/chubo/cmd/chuboctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
