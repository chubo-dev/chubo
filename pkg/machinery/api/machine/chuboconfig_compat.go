// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package machine

// GetChuboconfig returns the generated client config bytes using a chubo-primary accessor.
func (x *GenerateClientConfiguration) GetChuboconfig() []byte {
	if x == nil {
		return nil
	}

	return x.Talosconfig
}

// SetChuboconfig stores generated client config bytes using a chubo-primary mutator.
func (x *GenerateClientConfiguration) SetChuboconfig(config []byte) {
	if x == nil {
		return
	}

	x.Talosconfig = config
}
