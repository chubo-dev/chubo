// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package v1alpha1_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chubo-dev/chubo/pkg/machinery/config/generate"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func TestRedactSecrets(t *testing.T) {
	input, err := generate.NewInput("test", "https://doesntmatter:6443", constants.DefaultKubernetesVersion)
	require.NoError(t, err)

	container, err := input.Config(machine.TypeControlPlane)
	if err != nil {
		return
	}

	config := container.RawV1Alpha1()

	require.NotEmpty(t, config.MachineConfig.MachineToken)
	require.NotEmpty(t, config.MachineConfig.MachineCA.Key)
	require.NotEmpty(t, config.ClusterConfig.ClusterSecret)

	replacement := "**.***"

	config.Redact(replacement)

	require.Equal(t, replacement, config.Machine().Security().Token())
	require.Equal(t, replacement, string(config.Machine().Security().IssuingCA().Key))
	require.Equal(t, replacement, config.Cluster().Secret())
}
