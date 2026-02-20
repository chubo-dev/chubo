// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gen

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"

	"github.com/chubo-dev/chubo/pkg/cli"
	"github.com/chubo-dev/chubo/pkg/machinery/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config/configloader"
	"github.com/chubo-dev/chubo/pkg/machinery/config/generate/secrets"
)

var genSecretsCmdFlags struct {
	outputFile             string
	chuboVersion           string
	fromControlplaneConfig string
}

const (
	genSecretsChuboVersionFlagName = "chubo-version"
	legacySecretsVersionAliasFlag  = "talos-version"
)

// genSecretsCmd represents the `gen secrets` command.
var genSecretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Generates a secrets bundle file which can later be used to generate a config",
	Long:  ``,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			secretsBundle   *secrets.Bundle
			versionContract *config.VersionContract
			err             error
		)

		if genSecretsCmdFlags.chuboVersion != "" {
			versionContract, err = config.ParseContractFromVersion(genSecretsCmdFlags.chuboVersion)
			if err != nil {
				return fmt.Errorf("invalid chubo-version: %w", err)
			}
		}

		switch {
		case genSecretsCmdFlags.fromControlplaneConfig != "":
			var cfg config.Provider

			cfg, err = configloader.NewFromFile(genSecretsCmdFlags.fromControlplaneConfig)
			if err != nil {
				return fmt.Errorf("failed to load controlplane config: %w", err)
			}

			secretsBundle = secrets.NewBundleFromConfig(secrets.NewFixedClock(time.Now()), cfg)
		default:
			secretsBundle, err = secrets.NewBundle(secrets.NewFixedClock(time.Now()),
				versionContract,
			)
		}

		if err != nil {
			return fmt.Errorf("failed to create secrets bundle: %w", err)
		}

		return writeSecretsBundleToFile(secretsBundle)
	},
}

func writeSecretsBundleToFile(bundle *secrets.Bundle) error {
	bundleBytes, err := yaml.Marshal(bundle)
	if err != nil {
		return err
	}

	if genSecretsCmdFlags.outputFile == stdoutOutput {
		_, err = os.Stdout.Write(bundleBytes)

		return err
	}

	if err = validateFileExists(genSecretsCmdFlags.outputFile); err != nil {
		return err
	}

	return os.WriteFile(genSecretsCmdFlags.outputFile, bundleBytes, 0o600)
}

func init() {
	genSecretsCmd.Flags().StringVarP(&genSecretsCmdFlags.outputFile, "output-file", "o", "secrets.yaml", `path of the output file, or "-" for stdout`)
	genSecretsCmd.Flags().StringVar(&genSecretsCmdFlags.chuboVersion, genSecretsChuboVersionFlagName, "", "the desired Chubo OS version to generate secrets bundle for (backwards compatibility, e.g. v0.8)")
	genSecretsCmd.Flags().StringVar(&genSecretsCmdFlags.chuboVersion, legacySecretsVersionAliasFlag, "", fmt.Sprintf("Legacy alias for --%s.", genSecretsChuboVersionFlagName))
	cli.Should(genSecretsCmd.Flags().MarkHidden(legacySecretsVersionAliasFlag))
	genSecretsCmd.Flags().StringVar(&genSecretsCmdFlags.fromControlplaneConfig, "from-controlplane-config", "", "use the provided control-plane machine configuration as input")

	Cmd.AddCommand(genSecretsCmd)
}
