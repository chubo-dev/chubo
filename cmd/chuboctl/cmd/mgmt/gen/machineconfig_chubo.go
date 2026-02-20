// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siderolabs/go-pointer"
	"github.com/spf13/cobra"

	"github.com/chubo-dev/chubo/pkg/machinery/config/encoder"
	"github.com/chubo-dev/chubo/pkg/machinery/config/generate/secrets"
	chubotypes "github.com/chubo-dev/chubo/pkg/machinery/config/types/chubo"
)

var genMachineConfigFlags struct {
	output          string
	id              string
	installDisk     string
	installImage    string
	wipe            bool
	registryMirrors []string
	withSecrets     string

	withChubo             bool
	chuboRole             string
	openWontonArtifactURL string
	openGyozaArtifactURL  string
	chuboBootstrapExpect  int
	chuboJoin             []string

	withOpenBao bool
	openBaoMode string
}

func newMachineConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "machineconfig",
		Short: "Generate a minimal machine config for Chubo",
		Long: `Generates a single YAML document:

  apiVersion: chubo.dev/v1alpha1
  kind: MachineConfig

The output is suitable for ` + "`chuboctl apply-config`" + ` in the ` + "`chubo`" + ` build variant.
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if genMachineConfigFlags.output == "" {
				genMachineConfigFlags.output = stdoutOutput
			}

			var bundle *secrets.Bundle
			var err error

			switch strings.TrimSpace(genMachineConfigFlags.withSecrets) {
			case "":
				bundle, err = secrets.NewBundle(secrets.NewClock(), nil)
				if err != nil {
					return fmt.Errorf("failed to generate secrets bundle: %w", err)
				}
			default:
				bundle, err = secrets.LoadBundle(genMachineConfigFlags.withSecrets)
				if err != nil {
					return fmt.Errorf("failed to load secrets bundle: %w", err)
				}
			}

			if bundle.Certs == nil || bundle.Certs.OS == nil {
				return errors.New("secrets bundle is missing OS CA (certs.os)")
			}

			if bundle.TrustdInfo == nil || strings.TrimSpace(bundle.TrustdInfo.Token) == "" {
				return errors.New("secrets bundle is missing trustd token (trustdinfo.token)")
			}

			mc := chubotypes.NewMachineConfigV1Alpha1()
			mc.Metadata.ID = strings.TrimSpace(genMachineConfigFlags.id)

			mc.Spec.Install = &chubotypes.InstallSpec{
				Disk:  strings.TrimSpace(genMachineConfigFlags.installDisk),
				Image: strings.TrimSpace(genMachineConfigFlags.installImage),
				Wipe:  pointer.To(genMachineConfigFlags.wipe),
			}

			// Keep Talos default NTP server unless explicitly overridden later.
			mc.Spec.Time = &chubotypes.TimeSpec{
				Servers: []string{"time.cloudflare.com"},
			}

			mc.Spec.Trust = &chubotypes.TrustSpec{
				Token: bundle.TrustdInfo.Token,
				CA: &chubotypes.CASpec{
					Crt: string(bundle.Certs.OS.Crt),
					Key: string(bundle.Certs.OS.Key),
				},
			}

			if genMachineConfigFlags.withChubo {
				role := strings.TrimSpace(genMachineConfigFlags.chuboRole)
				if role == "" {
					role = "server"
				}

				var bootstrapExpect *int
				if genMachineConfigFlags.chuboBootstrapExpect >= 0 {
					bootstrapExpect = pointer.To(genMachineConfigFlags.chuboBootstrapExpect)
				}

				join := make([]string, 0, len(genMachineConfigFlags.chuboJoin))
				for _, entry := range genMachineConfigFlags.chuboJoin {
					entry = strings.TrimSpace(entry)
					if entry == "" {
						continue
					}

					join = append(join, entry)
				}

				if len(join) == 0 {
					join = nil
				}

				mc.Spec.Modules = &chubotypes.ModulesSpec{
					Chubo: &chubotypes.ChuboModuleSpec{
						Enabled: pointer.To(true),
						Nomad: &chubotypes.ChuboRoleSpec{
							Enabled:         pointer.To(true),
							Role:            role,
							ArtifactURL:     strings.TrimSpace(genMachineConfigFlags.openWontonArtifactURL),
							BootstrapExpect: bootstrapExpect,
							Join:            join,
						},
						Consul: &chubotypes.ChuboRoleSpec{
							Enabled:         pointer.To(true),
							Role:            role,
							ArtifactURL:     strings.TrimSpace(genMachineConfigFlags.openGyozaArtifactURL),
							BootstrapExpect: bootstrapExpect,
							Join:            join,
						},
					},
				}

				if genMachineConfigFlags.withOpenBao {
					mode := strings.TrimSpace(genMachineConfigFlags.openBaoMode)
					if mode == "" {
						mode = "nomadJob"
					}

					mc.Spec.Modules.Chubo.OpenBao = &chubotypes.ChuboOpenBaoSpec{
						Enabled: pointer.To(true),
						Mode:    mode,
					}
				}
			}

			if len(genMachineConfigFlags.registryMirrors) > 0 {
				mc.Spec.Registry = &chubotypes.RegistrySpec{
					Mirrors: map[string]chubotypes.RegistryMirrorSpec{},
				}

				for _, spec := range genMachineConfigFlags.registryMirrors {
					left, right, ok := strings.Cut(spec, "=")
					if !ok {
						return fmt.Errorf("invalid registry mirror spec: %q", spec)
					}

					host := strings.TrimSpace(left)
					endpoint := strings.TrimSpace(right)

					if host == "" || endpoint == "" {
						return fmt.Errorf("invalid registry mirror spec: %q", spec)
					}

					m := mc.Spec.Registry.Mirrors[host]
					m.Endpoints = append(m.Endpoints, endpoint)
					mc.Spec.Registry.Mirrors[host] = m
				}
			}

			out, err := encoder.NewEncoder(mc, encoder.WithComments(encoder.CommentsDisabled)).Encode()
			if err != nil {
				return err
			}

			if genMachineConfigFlags.output == stdoutOutput {
				_, err = cmd.OutOrStdout().Write(out)
				return err
			}

			if err := validateFileExists(genMachineConfigFlags.output); err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(genMachineConfigFlags.output), 0o755); err != nil {
				return err
			}

			return os.WriteFile(genMachineConfigFlags.output, out, 0o600)
		},
	}

	cmd.Flags().StringVarP(&genMachineConfigFlags.output, "output", "o", stdoutOutput, `output path, or "-" for stdout`)
	cmd.Flags().StringVar(&genMachineConfigFlags.id, "id", "", "optional stable node id (metadata.id)")
	cmd.Flags().StringVar(&genMachineConfigFlags.installDisk, "install-disk", "/dev/sda", "disk to install to")
	cmd.Flags().StringVar(&genMachineConfigFlags.installImage, "install-image", "", "installer image to install from (leave empty if you set it via boot args)")
	cmd.Flags().BoolVar(&genMachineConfigFlags.wipe, "wipe", true, "wipe the install disk before installing")
	cmd.Flags().StringSliceVar(&genMachineConfigFlags.registryMirrors, "registry-mirror", nil, "registry mirrors in format: <registry host>=<mirror URL>")
	cmd.Flags().StringVar(&genMachineConfigFlags.withSecrets, "with-secrets", "", "use a secrets file generated using 'gen secrets' (optional)")

	cmd.Flags().BoolVar(&genMachineConfigFlags.withChubo, "with-chubo", false, "enable modules.chubo with openwonton/opengyoza defaults")
	cmd.Flags().StringVar(&genMachineConfigFlags.chuboRole, "chubo-role", "server", "chubo role for openwonton/opengyoza (server|client)")
	cmd.Flags().StringVar(&genMachineConfigFlags.openWontonArtifactURL, "openwonton-artifact-url", "", "override openwonton artifact URL (http(s)://...)")
	cmd.Flags().StringVar(&genMachineConfigFlags.openGyozaArtifactURL, "opengyoza-artifact-url", "", "override opengyoza artifact URL (http(s)://...)")
	cmd.Flags().IntVar(&genMachineConfigFlags.chuboBootstrapExpect, "chubo-bootstrap-expect", -1, "bootstrap_expect for openwonton/opengyoza (unset by default)")
	cmd.Flags().StringSliceVar(&genMachineConfigFlags.chuboJoin, "chubo-join", nil, "peer addresses to join/retry-join for openwonton/opengyoza")
	cmd.Flags().BoolVar(&genMachineConfigFlags.withOpenBao, "with-openbao", false, "enable modules.chubo.openbao (Nomad job controller)")
	cmd.Flags().StringVar(&genMachineConfigFlags.openBaoMode, "openbao-mode", "nomadJob", "openbao mode when enabled (nomadJob)")

	return cmd
}

func init() {
	Cmd.AddCommand(newMachineConfigCmd())
}
