// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package talos17 provides compatibility constants for Chubo 1.7.
package talos17

import (
	"github.com/blang/semver/v4"
)

// MajorMinor is the major.minor version of Chubo 1.7.
var MajorMinor = [2]uint64{1, 7}

// MinimumHostUpgradeVersion is the minimum version of Chubo that can be upgraded to 1.7.
var MinimumHostUpgradeVersion = semver.MustParse("1.4.0")

// MaximumHostDowngradeVersion is the maximum (not inclusive) version of Chubo that can be downgraded to 1.7.
var MaximumHostDowngradeVersion = semver.MustParse("1.9.0")

// DeniedHostUpgradeVersions are the versions of Chubo that cannot be upgraded to 1.7.
var DeniedHostUpgradeVersions []semver.Version
