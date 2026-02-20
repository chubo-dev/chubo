// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package common provides common functionality for chuboctl commands.
package common

import legacycommon "github.com/chubo-dev/chubo/cmd/talosctl/cmd/common"

// SuppressErrors reports whether command-level error printing should be suppressed.
func SuppressErrors() bool {
	return legacycommon.SuppressErrors
}

// SetSuppressErrors toggles command-level error printing.
func SetSuppressErrors(v bool) {
	legacycommon.SuppressErrors = v
}
