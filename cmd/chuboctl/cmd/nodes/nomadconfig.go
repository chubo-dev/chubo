// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package nodes

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chubo-dev/chubo/cmd/chuboctl/pkg/nodes/helpers"
	"github.com/chubo-dev/chubo/pkg/machinery/client"
)

var nomadConfigFlags struct {
	force bool
}

var nomadConfigCmd = &cobra.Command{
	Use:   "nomadconfig [local-path]",
	Short: "Download the Nomad client configuration bundle from the node",
	Long: `Download the Nomad client configuration bundle from the node.

By default the bundle is extracted to PWD/nomadconfig/.
If [local-path] is a directory, bundle is extracted under [local-path]/nomadconfig/.
If [local-path] does not exist, it is created and used as the extraction directory.
If [local-path] is "-", the raw .tar.gz bundle is written to stdout.`,
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

			bundle, err := downloadBundle(ctx, c.NomadConfigRaw)
			if err != nil {
				return err
			}

			return writeConfigBundle(localPath, bundle, "nomadconfig", nomadConfigFlags.force)
		})
	},
}

func init() {
	nomadConfigCmd.Flags().BoolVarP(&nomadConfigFlags.force, "force", "f", false, "Force overwrite if the output file already exists")
	addCommand(nomadConfigCmd)
}
