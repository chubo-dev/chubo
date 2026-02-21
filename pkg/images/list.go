// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package images

import (
	"github.com/google/go-containerregistry/pkg/name"
)

func mustParseReferenceWithTag(ref, tag string) name.Tag {
	r, err := name.ParseReference(ref)
	if err != nil {
		panic(err)
	}

	return r.Context().Tag(tag)
}

// TalosBundle holds the core images (and their versions) that are used to build Chubo.
type TalosBundle struct {
	Installer     name.Tag
	InstallerBase name.Tag
	Imager        name.Tag
	Talos         name.Tag
	TalosctlAll   name.Tag

	Overlays   name.Tag
	Extensions name.Tag
}

// ListSourcesFor returns source bundle for specific version.
func ListSourcesFor(tag string) TalosBundle {
	var bundle TalosBundle

	bundle.Installer = mustParseReferenceWithTag(DefaultInstallerImageRepository, tag)
	bundle.InstallerBase = mustParseReferenceWithTag(DefaultInstallerBaseImageRepository, tag)
	bundle.Imager = mustParseReferenceWithTag(DefaultImagerImageRepository, tag)
	bundle.Talos = mustParseReferenceWithTag(DefaultChuboImageRepository, tag)
	bundle.TalosctlAll = mustParseReferenceWithTag(DefaultChuboctlAllImageRepository, tag)

	bundle.Overlays = mustParseReferenceWithTag(DefaultOverlaysManifestRepository, tag)
	bundle.Extensions = mustParseReferenceWithTag(DefaultExtensionsManifestRepository, tag)

	return bundle
}
