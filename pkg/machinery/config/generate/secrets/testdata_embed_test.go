// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package secrets_test

import _ "embed"

var (
	//go:embed testdata/invalid-secrets.yaml
	invalidSecrets []byte
	//go:embed testdata/secrets.yaml
	validSecrets []byte
)

