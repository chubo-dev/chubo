// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package workload provides a source-clean alias over legacy CRI registry/image-cache resources.
package workload

import (
	"context"

	"github.com/cosi-project/runtime/pkg/state"

	criresources "github.com/chubo-dev/chubo/pkg/machinery/resources/cri"
)

type (
	RegistryTLSConfigExtended = criresources.RegistryTLSConfigExtended
	Registries                = criresources.Registries

	RegistriesConfig          = criresources.RegistriesConfig
	RegistriesConfigSpec      = criresources.RegistriesConfigSpec
	RegistriesConfigExtension = criresources.RegistriesConfigExtension

	RegistryMirrorConfig   = criresources.RegistryMirrorConfig
	RegistryEndpointConfig = criresources.RegistryEndpointConfig
	RegistryAuthConfig     = criresources.RegistryAuthConfig
	RegistryTLSConfig      = criresources.RegistryTLSConfig

	ImageCacheConfig          = criresources.ImageCacheConfig
	ImageCacheConfigSpec      = criresources.ImageCacheConfigSpec
	ImageCacheConfigExtension = criresources.ImageCacheConfigExtension
	ImageCacheStatus          = criresources.ImageCacheStatus
	ImageCacheCopyStatus      = criresources.ImageCacheCopyStatus

	SeccompProfile          = criresources.SeccompProfile
	SeccompProfileSpec      = criresources.SeccompProfileSpec
	SeccompProfileExtension = criresources.SeccompProfileExtension
)

const (
	NamespaceName       = criresources.NamespaceName
	RegistriesConfigType = criresources.RegistriesConfigType
	RegistriesConfigID   = criresources.RegistriesConfigID

	ImageCacheConfigType = criresources.ImageCacheConfigType
	ImageCacheConfigID   = criresources.ImageCacheConfigID

	ImageCacheStatusUnknown   = criresources.ImageCacheStatusUnknown
	ImageCacheStatusDisabled  = criresources.ImageCacheStatusDisabled
	ImageCacheStatusPreparing = criresources.ImageCacheStatusPreparing
	ImageCacheStatusReady     = criresources.ImageCacheStatusReady

	ImageCacheCopyStatusUnknown = criresources.ImageCacheCopyStatusUnknown
	ImageCacheCopyStatusSkipped = criresources.ImageCacheCopyStatusSkipped
	ImageCacheCopyStatusPending = criresources.ImageCacheCopyStatusPending
	ImageCacheCopyStatusReady   = criresources.ImageCacheCopyStatusReady

	SeccompProfileType = criresources.SeccompProfileType
)

func NewRegistriesConfig() *RegistriesConfig {
	return criresources.NewRegistriesConfig()
}

func NewImageCacheConfig() *ImageCacheConfig {
	return criresources.NewImageCacheConfig()
}

func NewSeccompProfile(id string) *SeccompProfile {
	return criresources.NewSeccompProfile(id)
}

func RegistryBuilder(st state.State) func(ctx context.Context) (Registries, error) {
	return criresources.RegistryBuilder(st)
}

func WaitForImageCache(ctx context.Context, st state.State) error {
	return criresources.WaitForImageCache(ctx, st)
}

func WaitForImageCacheCopy(ctx context.Context, st state.State) error {
	return criresources.WaitForImageCacheCopy(ctx, st)
}
