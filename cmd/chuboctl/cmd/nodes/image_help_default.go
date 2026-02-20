// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package nodes

func imageNamespaceHelp() string {
	return "namespace to use: `system` (core system images), `cri` (workload runtime images), or `inmem` (in-memory containerd instance)"
}

func imageCacheMirrorDefaults() []string {
	return []string{"docker.io", "ghcr.io"}
}

func registerImageBundleCommand() {
	imageCmd.AddCommand(imageTalosBundleCmd)
	imageTalosBundleCmd.PersistentFlags().BoolVar(&imageTalosBundleCmdFlags.overlays, "overlays", true, "Include images that belong to OS overlays")
	imageTalosBundleCmd.PersistentFlags().BoolVar(&imageTalosBundleCmdFlags.extensions, "extensions", true, "Include images that belong to OS extensions")
}
