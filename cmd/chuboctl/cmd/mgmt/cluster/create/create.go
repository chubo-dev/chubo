// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package create keeps cluster/create command registration side effects wired
// while command paths are migrated to cmd/chuboctl.
package create

import (
	_ "github.com/chubo-dev/chubo/cmd/talosctl/cmd/mgmt/cluster/create"
)
