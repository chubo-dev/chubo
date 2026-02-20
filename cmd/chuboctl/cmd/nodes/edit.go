//go:build !chubo

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package nodes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/chubo-dev/chubo/cmd/chuboctl/pkg/nodes/helpers"
	"github.com/chubo-dev/chubo/cmd/chuboctl/pkg/nodes/yamlstrip"
	"github.com/chubo-dev/chubo/pkg/machinery/api/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/client"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
)

var editCmdFlags struct {
	helpers.Mode

	namespace        string
	dryRun           bool
	configTryTimeout time.Duration
}

type editorLauncher struct {
	env []string
}

func newEditorLauncher(env []string) *editorLauncher {
	return &editorLauncher{
		env: env,
	}
}

func (l *editorLauncher) editorCommand() string {
	for _, key := range l.env {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	if runtime.GOOS == "windows" {
		return "notepad"
	}

	return "vi"
}

func (l *editorLauncher) LaunchTempFile(prefix, suffix string, data io.Reader) ([]byte, string, error) {
	file, err := os.CreateTemp("", prefix+"*"+suffix)
	if err != nil {
		return nil, "", err
	}

	path := file.Name()

	if _, err = io.Copy(file, data); err != nil {
		file.Close() //nolint:errcheck

		return nil, path, err
	}

	if err = file.Close(); err != nil {
		return nil, path, err
	}

	command := strings.Fields(l.editorCommand())
	if len(command) == 0 {
		return nil, path, errors.New("editor command is empty")
	}

	command = append(command, path)

	cmd := exec.Command(command[0], command[1:]...) //nolint:gosec
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err = cmd.Run(); err != nil {
		return nil, path, err
	}

	edited, err := os.ReadFile(path)
	if err != nil {
		return nil, path, err
	}

	return edited, path, nil
}

//nolint:gocyclo
func editFn(c *client.Client) func(context.Context, string, resource.Resource, error) error {
	var (
		path      string
		lastError string
	)

	edit := newEditorLauncher([]string{
		"CHUBO_EDITOR",
		"TALOS_EDITOR",
		"EDITOR",
	})

	return func(ctx context.Context, node string, mc resource.Resource, callError error) error {
		if callError != nil {
			return fmt.Errorf("%s: %w", node, callError)
		}

		if mc.Metadata().Type() != config.MachineConfigType {
			return errors.New("only the machineconfig resource can be edited")
		}

		id := mc.Metadata().ID()

		if id != config.ActiveID {
			return nil
		}

		body, err := extractMachineConfigBody(mc)
		if err != nil {
			return err
		}

		edited := body

		for {
			var buf bytes.Buffer

			w := io.Writer(&buf)

			_, err := fmt.Fprintf(w,
				"# Editing %s/%s at node %s\n", mc.Metadata().Type(), id, node,
			)
			if err != nil {
				return err
			}

			if lastError != "" {
				_, err = w.Write([]byte(addEditingComment(lastError)))
				if err != nil {
					return err
				}
			}

			_, err = w.Write(edited)
			if err != nil {
				return err
			}

			editedDiff := edited

			edited, path, err = edit.LaunchTempFile(fmt.Sprintf("%s-%s-edit-", mc.Metadata().Type(), id), ".yaml", &buf)
			if err != nil {
				return err
			}

			defer os.Remove(path) //nolint:errcheck

			edited = stripEditingComment(edited)

			// If we're retrying the loop because of an error, and no change was made in the file, short-circuit
			if lastError != "" && bytes.Equal(yamlstrip.Comments(editedDiff), yamlstrip.Comments(edited)) {
				if _, err = os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
					message := addEditingComment(lastError)
					message += fmt.Sprintf("A copy of your changes has been stored to %q\nEdit canceled, no valid changes were saved.\n", path)

					return errors.New(message)
				}
			}

			if len(bytes.TrimSpace(bytes.TrimSpace(yamlstrip.Comments(edited)))) == 0 {
				fmt.Fprintln(os.Stderr, "Apply was skipped: empty file.")

				break
			}

			if bytes.Equal(edited, body) {
				fmt.Fprintln(os.Stderr, "Apply was skipped: no changes detected.")

				break
			}

			resp, err := c.ApplyConfiguration(ctx, &machine.ApplyConfigurationRequest{
				Data:           edited,
				Mode:           editCmdFlags.Mode.Mode,
				DryRun:         editCmdFlags.dryRun,
				TryModeTimeout: durationpb.New(editCmdFlags.configTryTimeout),
			})
			if err != nil {
				lastError = err.Error()

				continue
			}

			helpers.PrintApplyResults(resp)

			break
		}

		return nil
	}
}

func stripEditingComment(in []byte) []byte {
	// Note: on Windows, we use crlf.NewCRLFWriter which converts LF to CRLF before opening the editor.
	// So this code block below undoes that conversion back to LF for consistent processing.
	if runtime.GOOS == "windows" {
		// revert back CRLF to LF for processing
		in = bytes.ReplaceAll(in, []byte("\r\n"), []byte("\n"))
	}

	for {
		idx := bytes.Index(in, []byte{'\n'})
		if idx == -1 {
			return in
		}

		if !bytes.HasPrefix(in, []byte("# ")) {
			return in
		}

		in = in[idx+1:]
	}
}

func addEditingComment(in string) string {
	lines := strings.Split(in, "\n")

	return fmt.Sprintf("# \n# %s\n", strings.Join(lines, "\n# "))
}

// editCmd represents the edit command.
var editCmd = &cobra.Command{
	Use:   "edit machineconfig",
	Short: "Edit machine configuration with the default editor.",
	Args:  cobra.RangeArgs(1, 2),
	Long: `The edit command allows you to directly edit the machine configuration
of a Chubo OS node using your preferred text editor.

It will open the editor defined by your CHUBO_EDITOR,
TALOS_EDITOR, or EDITOR environment variables, or fall back to 'vi' for Linux
or 'notepad' for Windows.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return WithClient(func(ctx context.Context, c *client.Client) error {
			if err := helpers.ClientVersionCheck(ctx, c); err != nil {
				return err
			}

			for _, node := range GlobalArgs.Nodes {
				nodeCtx := client.WithNodes(ctx, node)
				if err := helpers.ForEachResource(nodeCtx, c, nil, editFn(c), editCmdFlags.namespace, args...); err != nil {
					return err
				}
			}

			return nil
		})
	},
}

func init() {
	editCmd.Flags().StringVar(&editCmdFlags.namespace, "namespace", "", "resource namespace (default is to use default namespace per resource)")
	helpers.AddModeFlags(&editCmdFlags.Mode, editCmd)
	editCmd.Flags().BoolVar(&editCmdFlags.dryRun, "dry-run", false, "do not apply the change after editing and print the change summary instead")
	editCmd.Flags().DurationVar(&editCmdFlags.configTryTimeout, "timeout", constants.ConfigTryTimeout, "the config will be rolled back after specified timeout (if try mode is selected)")
	addCommand(editCmd)
}
