// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chubo-dev/chubo/pkg/machinery/config/generate"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func TestClusterDiscoveryDisabledByDefault(t *testing.T) {
	t.Parallel()

	input, err := generate.NewInput("test", "https://10.0.1.5", constants.DefaultWorkloadVersion)
	require.NoError(t, err)

	cfg, err := input.Config(machine.TypeControlPlane)
	require.NoError(t, err)

	raw := cfg.RawV1Alpha1()
	require.NotNil(t, raw.ClusterConfig)
	require.NotNil(t, raw.ClusterConfig.ClusterDiscoveryConfig)
	require.NotNil(t, raw.ClusterConfig.ClusterDiscoveryConfig.DiscoveryEnabled)

	assert.False(t, *raw.ClusterConfig.ClusterDiscoveryConfig.DiscoveryEnabled)
	assert.Empty(t, raw.ClusterConfig.ClusterDiscoveryConfig.DiscoveryRegistries.RegistryService.RegistryEndpoint)
}
