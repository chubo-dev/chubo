// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build integration

// Package integration_test contains core runners for integration tests
package integration_test

import (
	"github.com/stretchr/testify/suite"

	"github.com/chubo-dev/chubo/internal/integration/base"
)

// VersionSuite.
type VersionSuite struct {
	suite.Suite
	base.ChuboSuite
}

func (suite *VersionSuite) SuiteName() string {
	return "VersionSuite"
}

func (suite *VersionSuite) TestExpectedVersion() {
	const versionRegex = `v([0-9]+)\.([0-9]+)\.([0-9]+)(-[0-9]+-[a-z]+\.[0-9]+)?(-.g[a-f0-9]+)?(-dirty)?`

	suite.Assert().Regexp(versionRegex, suite.Version)
}

func init() {
	allSuites = append(allSuites, new(VersionSuite))
}
