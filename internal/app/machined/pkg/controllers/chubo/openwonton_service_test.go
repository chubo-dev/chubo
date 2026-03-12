// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInjectOpenWontonVaultTokenAddsTokenLine(t *testing.T) {
	t.Parallel()

	cfg := `vault {
  enabled = true
  address = "http://127.0.0.1:8200"
  allow_unauthenticated = true
}

server {
  enabled = true
}
`

	updated, changed, err := injectOpenWontonVaultToken(cfg, "root-token")
	require.NoError(t, err)
	require.True(t, changed)
	require.Contains(t, updated, `  token = "root-token"`)
}

func TestInjectOpenWontonVaultTokenReplacesExistingToken(t *testing.T) {
	t.Parallel()

	cfg := `vault {
  enabled = true
  address = "http://127.0.0.1:8200"
  allow_unauthenticated = true
  token = "old-token"
}
`

	updated, changed, err := injectOpenWontonVaultToken(cfg, "new-token")
	require.NoError(t, err)
	require.True(t, changed)
	require.Contains(t, updated, `  token = "new-token"`)
	require.NotContains(t, updated, `  token = "old-token"`)
}

func TestInjectOpenWontonVaultTokenIsNoopWhenTokenMatches(t *testing.T) {
	t.Parallel()

	cfg := `vault {
  enabled = true
  address = "http://127.0.0.1:8200"
  allow_unauthenticated = true
  token = "same-token"
}
`

	updated, changed, err := injectOpenWontonVaultToken(cfg, "same-token")
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, cfg, updated)
}
