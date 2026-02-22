// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package constants

import (
	"os"
	"strings"
)

const (
	// ChuboDiscoveryServiceEndpointEnvVar overrides the default discovery endpoint.
	ChuboDiscoveryServiceEndpointEnvVar = "CHUBO_DISCOVERY_SERVICE_ENDPOINT"

	// TalosDiscoveryServiceEndpointEnvVar is a legacy alias for overriding the discovery endpoint.
	TalosDiscoveryServiceEndpointEnvVar = "TALOS_DISCOVERY_SERVICE_ENDPOINT"
)

// EffectiveDiscoveryServiceEndpoint resolves the discovery endpoint from environment overrides.
func EffectiveDiscoveryServiceEndpoint() string {
	return firstNonEmptyEnv(DefaultDiscoveryServiceEndpoint, ChuboDiscoveryServiceEndpointEnvVar, TalosDiscoveryServiceEndpointEnvVar)
}

func firstNonEmptyEnv(defaultValue string, envKeys ...string) string {
	for _, key := range envKeys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}

	return defaultValue
}
