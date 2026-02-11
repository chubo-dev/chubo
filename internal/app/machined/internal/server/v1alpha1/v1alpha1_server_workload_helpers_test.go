// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runtime

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreferredRoutedNodeIP(t *testing.T) {
	t.Parallel()

	ips := []netip.Addr{
		netip.MustParseAddr("10.0.2.15"),
		netip.MustParseAddr("192.168.0.160"),
	}

	for _, tt := range []struct {
		name        string
		authorities []string
		expected    string
		found       bool
	}{
		{
			name:        "exact_ipv4_with_port",
			authorities: []string{"192.168.0.160:50000"},
			expected:    "192.168.0.160",
			found:       true,
		},
		{
			name:        "ipv6_with_brackets_and_port",
			authorities: []string{"[fd00::10]:50000"},
			found:       false,
		},
		{
			name:        "unknown_and_empty_ignored",
			authorities: []string{"", "unknown", "  ", "192.168.0.160:50000"},
			expected:    "192.168.0.160",
			found:       true,
		},
		{
			name:        "non_matching_host_ignored",
			authorities: []string{"localhost:50000"},
			found:       false,
		},
		{
			name:        "multiple_values_first_matching_wins",
			authorities: []string{"10.5.0.1:50000", "10.0.2.15:50000", "192.168.0.160:50000"},
			expected:    "10.0.2.15",
			found:       true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			addr, ok := preferredRoutedNodeIP(ips, tt.authorities)
			require.Equal(t, tt.found, ok)

			if tt.found {
				require.Equal(t, tt.expected, addr.String())
			}
		})
	}
}
