// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package constants

import (
	"os"
	"strings"
)

const (
	// ImageFactoryEmptySchematicID is the ID of an empty image factory schematic.
	ImageFactoryEmptySchematicID = "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba"

	// ChuboImageFactoryURLEnvVar overrides the default image factory URL.
	ChuboImageFactoryURLEnvVar = "CHUBO_IMAGE_FACTORY_URL"

	// TalosImageFactoryURLEnvVar is a legacy alias for overriding the image factory URL.
	TalosImageFactoryURLEnvVar = "TALOS_IMAGE_FACTORY_URL"
)

var (
	// ImageFactoryURL is the default image factory endpoint used by chuboctl.
	ImageFactoryURL = "https://factory.talos.dev/"
)

func init() {
	if value := strings.TrimSpace(os.Getenv(ChuboImageFactoryURLEnvVar)); value != "" {
		ImageFactoryURL = value

		return
	}

	if value := strings.TrimSpace(os.Getenv(TalosImageFactoryURLEnvVar)); value != "" {
		ImageFactoryURL = value
	}
}
