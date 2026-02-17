// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package runtimecfg provides support for containerd registry auth configuration.
package runtimecfg

import (
	"bytes"
	"maps"
	"path/filepath"
	"slices"

	"github.com/containerd/containerd/v2/core/remotes/docker"
	"github.com/pelletier/go-toml/v2"

	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/workload"
)

// GenerateRegistryConfig returns the containerd registry auth snippet.
//
// Once containerd supports different way of supplying auth info, this should be updated.
func GenerateRegistryConfig(r workload.Registries) ([]byte, error) {
	var ctrdCfg Config

	ctrdCfg.Plugins.Images.Registry.ConfigPath = filepath.Join(constants.EtcRuntimeConfdPath, "hosts")
	ctrdCfg.Plugins.Images.Registry.Configs = make(map[string]RegistryConfig)

	for _, registryHost := range slices.Sorted(maps.Keys(r.Auths())) {
		authConfig := r.Auths()[registryHost]

		cfg := RegistryConfig{}
		cfg.Auth = &AuthConfig{
			Username:      authConfig.Username(),
			Password:      authConfig.Password(),
			Auth:          authConfig.Auth(),
			IdentityToken: authConfig.IdentityToken(),
		}

		configHost, _ := docker.DefaultHost(registryHost) //nolint:errcheck // doesn't return an error

		ctrdCfg.Plugins.Images.Registry.Configs[configHost] = cfg
	}

	var buf bytes.Buffer

	if err := toml.NewEncoder(&buf).SetIndentTables(true).Encode(&ctrdCfg); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
