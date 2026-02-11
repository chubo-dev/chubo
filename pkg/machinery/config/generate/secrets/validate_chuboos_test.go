// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo || chuboos

package secrets_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	var bundle secrets.Bundle
	require.NoError(t, yaml.Unmarshal(validSecrets, &bundle))
	require.NoError(t, bundle.Validate())

	var invalidBundle secrets.Bundle
	require.NoError(t, yaml.Unmarshal(invalidSecrets, &invalidBundle))
	require.EqualError(t, invalidBundle.Validate(), `4 errors occurred:
	* cluster.secret is required
	* one of [secrets.secretboxencryptionsecret, secrets.aescbcencryptionsecret] is required
	* trustdinfo is required
	* certs.os is invalid: unsupported key type: "CERTIFICATE"

`)
}

