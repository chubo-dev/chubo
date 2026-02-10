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

var consulConfigFlags struct {
	force bool
}

var consulConfigCmd = &cobra.Command{
	Use:   "consulconfig [local-path]",
	Short: "Download the Consul client configuration bundle from the node",
	Long: `Download the Consul client configuration bundle from the node.

By default the bundle is written to PWD as 'consul.env'.
If [local-path] is a directory, 'consul.env' is written under it.
If [local-path] is "-", the config is written to stdout.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return WithClient(func(ctx context.Context, c *client.Client) error {
			if err := helpers.FailIfMultiNodes(ctx, "consulconfig"); err != nil {
				return err
			}

			localPath, err := defaultPath(args)
			if err != nil {
				return err
			}

			data, err := downloadSingleFile(ctx, c.ConsulConfigRaw, "consul.env")
			if err != nil {
				return err
			}

			return writeConfigFile(localPath, data, "consul.env", consulConfigFlags.force)
		})
	},
}

func init() {
	consulConfigCmd.Flags().BoolVarP(&consulConfigFlags.force, "force", "f", false, "Force overwrite if the output file already exists")
	addCommand(consulConfigCmd)
}
