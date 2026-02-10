// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package talos

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/siderolabs/talos/cmd/talosctl/pkg/talos/helpers"
	"github.com/siderolabs/talos/pkg/machinery/client"
)

var nomadConfigFlags struct {
	force bool
}

var nomadConfigCmd = &cobra.Command{
	Use:   "nomadconfig [local-path]",
	Short: "Download the Nomad client configuration bundle from the node",
	Long: `Download the Nomad client configuration bundle from the node.

By default the bundle is written to PWD as 'nomad.env'.
If [local-path] is a directory, 'nomad.env' is written under it.
If [local-path] is "-", the config is written to stdout.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return WithClient(func(ctx context.Context, c *client.Client) error {
			if err := helpers.FailIfMultiNodes(ctx, "nomadconfig"); err != nil {
				return err
			}

			localPath, err := defaultPath(args)
			if err != nil {
				return err
			}

			data, err := downloadSingleFile(ctx, c.NomadConfigRaw, "nomad.env")
			if err != nil {
				return err
			}

			return writeConfigFile(localPath, data, "nomad.env", nomadConfigFlags.force)
		})
	},
}

func init() {
	nomadConfigCmd.Flags().BoolVarP(&nomadConfigFlags.force, "force", "f", false, "Force overwrite if the output file already exists")
	addCommand(nomadConfigCmd)
}
