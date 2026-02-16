// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha1

import (
	"github.com/siderolabs/go-pointer"

	"github.com/chubo-dev/chubo/pkg/machinery/config/config"
	"github.com/chubo-dev/chubo/pkg/machinery/nethelpers"
)

// DiskQuotaSupportEnabled implements config.Features interface.
func (f *FeaturesConfig) DiskQuotaSupportEnabled() bool {
	return pointer.SafeDeref(f.DiskQuotaSupport)
}

// HostDNS implements config.Features interface.
func (f *FeaturesConfig) HostDNS() config.HostDNS {
	if f.HostDNSSupport == nil {
		return &HostDNSConfig{}
	}

	return f.HostDNSSupport
}

// ImageCache implements config.Features interface.
func (f *FeaturesConfig) ImageCache() config.ImageCache {
	if f.ImageCacheSupport == nil {
		return &ImageCacheConfig{}
	}

	return f.ImageCacheSupport
}

// NodeAddressSortAlgorithm implements config.Features interface.
func (f *FeaturesConfig) NodeAddressSortAlgorithm() nethelpers.AddressSortAlgorithm {
	if f.FeatureNodeAddressSortAlgorithm == "" {
		return nethelpers.AddressSortAlgorithmV1
	}

	res, err := nethelpers.AddressSortAlgorithmString(f.FeatureNodeAddressSortAlgorithm)
	if err != nil {
		return nethelpers.AddressSortAlgorithmV1
	}

	return res
}

// Enabled implements config.HostDNS.
func (h *HostDNSConfig) Enabled() bool {
	return pointer.SafeDeref(h.HostDNSEnabled)
}

// ResolveMemberNames implements config.HostDNS.
func (h *HostDNSConfig) ResolveMemberNames() bool {
	return pointer.SafeDeref(h.HostDNSResolveMemberNames)
}

// LocalEnabled implements config.ImageCache.
func (i *ImageCacheConfig) LocalEnabled() bool {
	return pointer.SafeDeref(i.CacheLocalEnabled)
}
