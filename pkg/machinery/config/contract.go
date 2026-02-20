// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// VersionContract describes Chubo version to generate config for.
//
// Config generation only supports backwards compatibility (e.g. Chubo 0.9 can generate configs for Chubo 0.9 and 0.8).
// Matching version of the machinery package is required to generate configs for the current version of Chubo.
//
// Nil value of *VersionContract always describes current version of Chubo.
type VersionContract struct {
	Major int
	Minor int
}

// Well-known Chubo version contracts.
var (
	ChuboVersionCurrent = (*VersionContract)(nil)
	ChuboVersion1_13    = &VersionContract{1, 13}
	ChuboVersion1_12    = &VersionContract{1, 12}
	ChuboVersion1_11    = &VersionContract{1, 11}
	ChuboVersion1_10    = &VersionContract{1, 10}
	ChuboVersion1_9     = &VersionContract{1, 9}
	ChuboVersion1_8     = &VersionContract{1, 8}
	ChuboVersion1_7     = &VersionContract{1, 7}
	ChuboVersion1_6     = &VersionContract{1, 6}
	ChuboVersion1_5     = &VersionContract{1, 5}
	ChuboVersion1_4     = &VersionContract{1, 4}
	ChuboVersion1_3     = &VersionContract{1, 3}
	ChuboVersion1_2     = &VersionContract{1, 2}
	ChuboVersion1_1     = &VersionContract{1, 1}
	ChuboVersion1_0     = &VersionContract{1, 0}
)

// Legacy compatibility aliases.
var (
	TalosVersionCurrent = ChuboVersionCurrent
	TalosVersion1_13    = ChuboVersion1_13
	TalosVersion1_12    = ChuboVersion1_12
	TalosVersion1_11    = ChuboVersion1_11
	TalosVersion1_10    = ChuboVersion1_10
	TalosVersion1_9     = ChuboVersion1_9
	TalosVersion1_8     = ChuboVersion1_8
	TalosVersion1_7     = ChuboVersion1_7
	TalosVersion1_6     = ChuboVersion1_6
	TalosVersion1_5     = ChuboVersion1_5
	TalosVersion1_4     = ChuboVersion1_4
	TalosVersion1_3     = ChuboVersion1_3
	TalosVersion1_2     = ChuboVersion1_2
	TalosVersion1_1     = ChuboVersion1_1
	TalosVersion1_0     = ChuboVersion1_0
)

var versionRegexp = regexp.MustCompile(`^v(\d+)\.(\d+)($|\.)`)

// ParseContractFromVersion parses a Chubo version into VersionContract.
func ParseContractFromVersion(version string) (*VersionContract, error) {
	version = "v" + strings.TrimPrefix(version, "v")

	matches := versionRegexp.FindStringSubmatch(version)
	if len(matches) < 3 {
		return nil, fmt.Errorf("error parsing version %q", version)
	}

	var contract VersionContract

	contract.Major, _ = strconv.Atoi(matches[1]) //nolint:errcheck
	contract.Minor, _ = strconv.Atoi(matches[2]) //nolint:errcheck

	return &contract, nil
}

// String returns string representation of the contract.
func (contract *VersionContract) String() string {
	if contract == nil {
		return "current"
	}

	return fmt.Sprintf("v%d.%d", contract.Major, contract.Minor)
}

// Greater compares contract to another contract.
func (contract *VersionContract) Greater(other *VersionContract) bool {
	if contract == nil {
		return other != nil
	}

	if other == nil {
		return false
	}

	return contract.Major > other.Major || (contract.Major == other.Major && contract.Minor > other.Minor)
}

// StableHostnameEnabled returns true if stable hostname generation should be enabled by default.
func (contract *VersionContract) StableHostnameEnabled() bool {
	return contract.Greater(ChuboVersion1_1)
}

// ApidExtKeyUsageCheckEnabled returns true if apid should check ext key usage of client certificates.
func (contract *VersionContract) ApidExtKeyUsageCheckEnabled() bool {
	return contract.Greater(ChuboVersion1_2)
}

// SecretboxEncryptionSupported returns true if encryption with secretbox is supported.
func (contract *VersionContract) SecretboxEncryptionSupported() bool {
	return contract.Greater(ChuboVersion1_2)
}

// DiskQuotaSupportEnabled returns true if XFS filesystems should enable project quota.
func (contract *VersionContract) DiskQuotaSupportEnabled() bool {
	return contract.Greater(ChuboVersion1_4)
}

// SecureBootEnrollEnforcementSupported returns true if the Chubo version supports SecureBoot enforcement on enroll.
func (contract *VersionContract) SecureBootEnrollEnforcementSupported() bool {
	return contract.Greater(ChuboVersion1_7)
}

// VolumeConfigEncryptionSupported returns true if the Chubo version supports disk encryption via VolumeConfig.
func (contract *VersionContract) VolumeConfigEncryptionSupported() bool {
	return contract.Greater(ChuboVersion1_10)
}

// MultidocNetworkConfigSupported returns true if the Chubo version supports multiple NetworkConfig documents.
func (contract *VersionContract) MultidocNetworkConfigSupported() bool {
	return contract.Greater(ChuboVersion1_11)
}

// HideDeprecatedMachineFeatures returns true if deprecated machine feature flags should be hidden.
func (contract *VersionContract) HideDeprecatedMachineFeatures() bool {
	return contract.Greater(ChuboVersion1_11)
}

// PopulateEndpointSANsByDefault returns true if endpoint SANs should be derived from ControlPlaneEndpoint.
func (contract *VersionContract) PopulateEndpointSANsByDefault() bool {
	return !contract.Greater(ChuboVersion1_11)
}

// GrubUseUKICmdlineDefault returns true if the Chubo version should use UKI cmdline by default.
func (contract *VersionContract) GrubUseUKICmdlineDefault() bool {
	return contract.Greater(ChuboVersion1_11)
}
