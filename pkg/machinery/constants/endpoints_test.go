// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package constants

import "testing"

func TestEffectiveDiscoveryServiceEndpoint(t *testing.T) {
	t.Run("uses default when no env override is set", func(t *testing.T) {
		t.Setenv(ChuboDiscoveryServiceEndpointEnvVar, "")
		t.Setenv(TalosDiscoveryServiceEndpointEnvVar, "")

		if got := EffectiveDiscoveryServiceEndpoint(); got != DefaultDiscoveryServiceEndpoint {
			t.Fatalf("expected %q, got %q", DefaultDiscoveryServiceEndpoint, got)
		}
	})

	t.Run("prefers chubo env override", func(t *testing.T) {
		t.Setenv(ChuboDiscoveryServiceEndpointEnvVar, " https://discovery.chubo.dev/ ")
		t.Setenv(TalosDiscoveryServiceEndpointEnvVar, "https://discovery.talos.dev/")

		if got := EffectiveDiscoveryServiceEndpoint(); got != "https://discovery.chubo.dev/" {
			t.Fatalf("expected chubo override, got %q", got)
		}
	})

	t.Run("falls back to legacy talos env override", func(t *testing.T) {
		t.Setenv(ChuboDiscoveryServiceEndpointEnvVar, "")
		t.Setenv(TalosDiscoveryServiceEndpointEnvVar, "https://discovery.legacy.example/")

		if got := EffectiveDiscoveryServiceEndpoint(); got != "https://discovery.legacy.example/" {
			t.Fatalf("expected legacy override, got %q", got)
		}
	})
}
