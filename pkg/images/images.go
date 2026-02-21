// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package images provides some default images.
package images

import (
	"github.com/chubo-dev/chubo/pkg/machinery/gendata"
	"github.com/chubo-dev/chubo/pkg/machinery/version"
)

var (
	// Username is the default registry username.
	Username = gendata.ImagesUsername

	// Registry is the default registry.
	Registry = gendata.ImagesRegistry

	// DefaultInstallerImageName is the default container image name for
	// the installer.
	DefaultInstallerImageName = Username + "/installer"

	// DefaultInstallerImageRepository is the default container repository for
	// the installer.
	DefaultInstallerImageRepository = Registry + "/" + DefaultInstallerImageName

	// DefaultInstallerImage is the default installer image.
	DefaultInstallerImage = DefaultInstallerImageRepository + ":" + version.Tag

	// DefaultChuboImageName is the default container image name for
	// the Chubo image.
	//
	// Legacy repository path is retained for compatibility.
	DefaultChuboImageName = Username + "/talos"

	// DefaultChuboImageRepository is the default container repository for
	// the Chubo image.
	DefaultChuboImageRepository = Registry + "/" + DefaultChuboImageName

	// DefaultChuboImage is the default Chubo image.
	DefaultChuboImage = DefaultChuboImageRepository + ":" + version.Tag

	// DefaultTalosImageName is a compatibility alias for DefaultChuboImageName.
	DefaultTalosImageName = DefaultChuboImageName
	// DefaultTalosImageRepository is a compatibility alias for DefaultChuboImageRepository.
	DefaultTalosImageRepository = DefaultChuboImageRepository
	// DefaultTalosImage is a compatibility alias for DefaultChuboImage.
	DefaultTalosImage = DefaultChuboImage

	// DefaultInstallerBaseImageRepository is the default container repository for
	// installer-base image.
	DefaultInstallerBaseImageRepository = Registry + "/" + Username + "/installer-base"

	// DefaultImagerImageRepository is the default container repository for
	// imager image.
	DefaultImagerImageRepository = Registry + "/" + Username + "/imager"

	// DefaultChuboctlAllImageRepository is the default container repository for
	// chuboctl-all image.
	//
	// Legacy repository path is retained for compatibility.
	DefaultChuboctlAllImageRepository = Registry + "/" + Username + "/talosctl-all"

	// DefaultTalosctlAllImageRepository is a compatibility alias for DefaultChuboctlAllImageRepository.
	DefaultTalosctlAllImageRepository = DefaultChuboctlAllImageRepository

	// DefaultOverlaysManifestName is the default container manifest name for
	// the overlays.
	DefaultOverlaysManifestName = Username + "/overlays"

	// DefaultOverlaysManifestRepository is the default container repository for
	// overlays manifest.
	DefaultOverlaysManifestRepository = Registry + "/" + DefaultOverlaysManifestName

	// DefaultExtensionsManifestName is the default container manifest name for
	// the extensions.
	DefaultExtensionsManifestName = Username + "/extensions"

	// DefaultExtensionsManifestRepository is the default container repository for
	// extensions manifest.
	DefaultExtensionsManifestRepository = Registry + "/" + DefaultExtensionsManifestName
)
