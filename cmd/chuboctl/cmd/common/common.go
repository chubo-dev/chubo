// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package common provides common functionality for chuboctl commands.
package common

var suppressErrors bool

// SuppressErrors reports whether command-level error printing should be suppressed.
func SuppressErrors() bool {
	return suppressErrors
}

// SetSuppressErrors toggles command-level error printing.
func SetSuppressErrors(v bool) {
	suppressErrors = v
}
