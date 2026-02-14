// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo

package v1alpha1

import "os"

// NodeName implements the Runtime interface.
func (r *Runtime) NodeName() (string, error) {
	return os.Hostname()
}
