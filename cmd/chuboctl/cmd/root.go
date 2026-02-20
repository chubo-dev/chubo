// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chubo-dev/chubo/cmd/chuboctl/cmd/common"
	"github.com/chubo-dev/chubo/cmd/chuboctl/cmd/mgmt"
	"github.com/chubo-dev/chubo/cmd/chuboctl/cmd/mgmt/cluster"
	_ "github.com/chubo-dev/chubo/cmd/chuboctl/cmd/mgmt/cluster/create" // import to get the command registered via the init() function.
	"github.com/chubo-dev/chubo/cmd/chuboctl/cmd/nodes"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "chuboctl",
	Short: "A CLI for out-of-band management of Chubo OS nodes",
	Long: `Manage Chubo OS nodes over the OS API.

The OS API is the only remote control plane.
Workload APIs (OpenWonton/OpenGyoza/OpenBao) are accessed via helper bundles:
nomadconfig, consulconfig, and openbaoconfig.`,
	SilenceErrors:     true,
	SilenceUsage:      true,
	DisableAutoGenTag: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	cmd, err := rootCmd.ExecuteContextC(context.Background())
	if err != nil && !common.SuppressErrors() {
		fmt.Fprintln(os.Stderr, err.Error())

		errorString := err.Error()
		// TODO: this is a nightmare, but arg-flag related validation returns simple `fmt.Errorf`, no way to distinguish
		//       these errors
		if strings.Contains(errorString, "arg(s)") || strings.Contains(errorString, "flag") || strings.Contains(errorString, "command") {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, cmd.UsageString())
		}
	}

	return err
}

func init() {
	const (
		nodesGroup   = "nodes"
		mgmtGroup    = "mgmt"
		clusterGroup = "cluster"
	)

	rootCmd.AddGroup(&cobra.Group{ID: nodesGroup, Title: "Manage running Chubo OS nodes:"})
	rootCmd.AddGroup(&cobra.Group{ID: mgmtGroup, Title: "Commands to generate and manage machine configuration offline:"})
	rootCmd.AddGroup(&cobra.Group{ID: clusterGroup, Title: "Local cluster commands:"})

	for _, cmd := range mgmt.Commands {
		cmd.GroupID = mgmtGroup
		if cmd == cluster.Cmd {
			cmd.GroupID = clusterGroup
		}

		rootCmd.AddCommand(cmd)
	}

	for _, cmd := range nodes.Commands {
		cmd.GroupID = nodesGroup
		rootCmd.AddCommand(cmd)
	}
}
