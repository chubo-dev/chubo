// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runtimecfg

// AuthConfig represents the registry auth options.
type AuthConfig struct {
	Username      string `toml:"username"`
	Password      string `toml:"password"`
	Auth          string `toml:"auth"`
	IdentityToken string `toml:"identitytoken"`
}

// RegistryConfig represents a registry.
type RegistryConfig struct {
	Auth *AuthConfig `toml:"auth"`
}

// Registry represents the registry configuration.
type Registry struct {
	ConfigPath string                    `toml:"config_path"`
	Configs    map[string]RegistryConfig `toml:"configs"`
}

// ImagePluginConfig represents the containerd image plugin config.
type RuntimeConfig struct {
	Registry Registry `toml:"registry"`
}

// PluginsConfig represents the containerd plugin configuration relevant to registry auth.
type PluginsConfig struct {
	Images RuntimeConfig `toml:"io.containerd.cri.v1.images"`
}

// Config represnts the containerd config.
type Config struct {
	Plugins PluginsConfig `toml:"plugins"`
}
