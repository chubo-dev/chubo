// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chubo-dev/chubo/pkg/cli"
)

// completionCmd represents the completion command.
var completionCmd = &cobra.Command{
	Use:   "completion SHELL",
	Short: "Output shell completion code for the specified shell (bash, fish or zsh)",
	Long: `Output shell completion code for the specified shell (bash, fish or zsh).
The shell code must be evaluated to provide interactive
completion of chuboctl commands.  This can be done by sourcing it from
the .bash_profile.

Note for zsh users: [1] zsh completions are only supported in versions of zsh >= 5.2`,
	Example: `# Installing bash completion on macOS using homebrew
## If running Bash 3.2 included with macOS
	brew install bash-completion
## or, if running Bash 4.1+
	brew install bash-completion@2
## If chuboctl is installed via homebrew, this should start working immediately.
## If you've installed via other means, you may need add the completion to your completion directory
	chuboctl completion bash > $(brew --prefix)/etc/bash_completion.d/chuboctl

# Installing bash completion on Linux
## If bash-completion is not installed on Linux, please install the 'bash-completion' package
## via your distribution's package manager.
## Load the chuboctl completion code for bash into the current shell
	source <(chuboctl completion bash)
## Write bash completion code to a file and source if from .bash_profile
	chuboctl completion bash > "${CHUBO_HOME:-$HOME/.chubo}/completion.bash.inc"
	printf '
		# chuboctl shell completion
		source "${CHUBO_HOME:-$HOME/.chubo}/completion.bash.inc"
		' >> $HOME/.bash_profile
	source $HOME/.bash_profile
# Load the chuboctl completion code for fish[1] into the current shell
	chuboctl completion fish | source
# Set the chuboctl completion code for fish[1] to autoload on startup
    chuboctl completion fish > ~/.config/fish/completions/chuboctl.fish
# Load the chuboctl completion code for zsh[1] into the current shell
	source <(chuboctl completion zsh)
# Set the chuboctl completion code for zsh[1] to autoload on startup
    chuboctl completion zsh > "${fpath[1]}/_chuboctl"`,
	ValidArgs: []string{"bash", "fish", "zsh"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			cli.Should(cmd.Usage())
			os.Exit(1)
		}

		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "zsh":
			err := rootCmd.GenZshCompletion(os.Stdout)
			// cobra does not hook the completion, so let's do it manually
			fmt.Printf("compdef _chuboctl chuboctl chuboctl")

			return err
		default:
			return fmt.Errorf("unsupported shell %q", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
