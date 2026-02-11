//go:build chubo

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package talos

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"

	commonapi "github.com/siderolabs/talos/pkg/machinery/api/common"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"github.com/siderolabs/talos/pkg/machinery/formatters"
	clusterresource "github.com/siderolabs/talos/pkg/machinery/resources/cluster"
)

var supportCmdFlags struct {
	output     string
	numWorkers int
	verbose    bool
}

// supportCmd represents the support command.
var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "Dump debug information about the cluster",
	Long: `Generated bundle contains the following debug information:

- For each node:
  - Kernel logs.
  - OS services state and logs.
  - Controller runtime dependency graph.
  - Mounts list.
  - Disk IO pressure snapshot.
  - Processes snapshot.
  - OS version summary.
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(GlobalArgs.Nodes) == 0 {
			return errors.New("please provide at least a single node to gather the debug information from")
		}

		f, err := openArchive()
		if err != nil {
			return err
		}
		defer f.Close() //nolint:errcheck

		zw := zip.NewWriter(f)
		defer zw.Close() //nolint:errcheck

		if err = collectData(zw); err != nil {
			return err
		}

		if err = zw.Close(); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Support bundle is written to %s\n", supportCmdFlags.output)

		return nil
	},
}

func collectData(zw *zip.Writer) error {
	return WithClientNoNodes(func(ctx context.Context, c *client.Client) error {
		for _, node := range GlobalArgs.Nodes {
			if supportCmdFlags.verbose {
				fmt.Fprintf(os.Stderr, "collecting support data for node %s\n", node)
			}

			nodeCtx := client.WithNode(ctx, node)

			if err := collectNodeData(nodeCtx, c, zw, node); err != nil {
				return err
			}
		}

		return nil
	})
}

func collectNodeData(ctx context.Context, c *client.Client, zw *zip.Writer, node string) error {
	write := func(name string, collect func() ([]byte, error)) error {
		data, err := collect()
		if err != nil {
			data = []byte(fmt.Sprintf("error: %s\n", err))
		}

		return writeZipEntry(zw, path.Join(node, name), data)
	}

	if err := write("summary", func() ([]byte, error) { return collectSummary(ctx, c) }); err != nil {
		return err
	}

	if err := write("dmesg.log", func() ([]byte, error) { return collectDmesg(ctx, c) }); err != nil {
		return err
	}

	if err := write("dependencies.dot", func() ([]byte, error) { return collectDependencies(ctx, c) }); err != nil {
		return err
	}

	if err := write("mounts", func() ([]byte, error) { return collectMounts(ctx, c) }); err != nil {
		return err
	}

	if err := write("io", func() ([]byte, error) { return collectIOPressure(ctx, c) }); err != nil {
		return err
	}

	if err := write("processes", func() ([]byte, error) { return collectProcesses(ctx, c) }); err != nil {
		return err
	}

	serviceIDs, serviceListData, err := collectServiceList(ctx, c)
	if err != nil {
		serviceListData = []byte(fmt.Sprintf("error: %s\n", err))
	}

	if err = writeZipEntry(zw, path.Join(node, "service-list.log"), serviceListData); err != nil {
		return err
	}

	for _, serviceID := range serviceIDs {
		id := serviceID

		if err = write(path.Join("service-logs", id+".state"), func() ([]byte, error) {
			return collectServiceState(ctx, c, id)
		}); err != nil {
			return err
		}

		if err = write(path.Join("service-logs", id+".log"), func() ([]byte, error) {
			return collectServiceLog(ctx, c, id)
		}); err != nil {
			return err
		}
	}

	return nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}

	_, err = w.Write(data)

	return err
}

func collectSummary(ctx context.Context, c *client.Client) ([]byte, error) {
	resp, err := c.Version(ctx)
	if err != nil {
		return nil, err
	}

	var b strings.Builder

	b.WriteString("Server:\n")

	for _, msg := range resp.Messages {
		if msg.Version == nil {
			continue
		}

		node := "<unknown>"
		if msg.Metadata != nil && msg.Metadata.Hostname != "" {
			node = msg.Metadata.Hostname
		}

		fmt.Fprintf(&b, "  Node: %s\n", node)
		fmt.Fprintf(&b, "  Tag: %s\n", msg.Version.Tag)
		fmt.Fprintf(&b, "  SHA: %s\n", msg.Version.Sha)
		fmt.Fprintf(&b, "  OS/Arch: %s/%s\n", msg.Version.Os, msg.Version.Arch)
	}

	return []byte(b.String()), nil
}

func collectDmesg(ctx context.Context, c *client.Client) ([]byte, error) {
	stream, err := c.Dmesg(ctx, false, false)
	if err != nil {
		return nil, err
	}

	return readDataStream(stream.Recv)
}

func collectDependencies(ctx context.Context, c *client.Client) ([]byte, error) {
	resp, err := c.Inspect.ControllerRuntimeDependencies(ctx)
	if err != nil && resp == nil {
		return nil, fmt.Errorf("error getting controller runtime dependencies: %w", err)
	}

	var b bytes.Buffer

	if err = formatters.RenderGraph(ctx, c, resp, &b, true); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func collectMounts(ctx context.Context, c *client.Client) ([]byte, error) {
	resp, err := c.Mounts(ctx)
	if err != nil && resp == nil {
		return nil, fmt.Errorf("error getting mounts: %w", err)
	}

	var b bytes.Buffer

	if err = formatters.RenderMounts(resp, &b, nil); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func collectIOPressure(ctx context.Context, c *client.Client) ([]byte, error) {
	resp, err := c.MachineClient.DiskStats(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer

	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tIO_TIME\tIO_TIME_WEIGHTED\tDISK_WRITE_SECTORS\tDISK_READ_SECTORS")

	for _, msg := range resp.Messages {
		for _, stat := range msg.Devices {
			fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\n",
				stat.Name,
				stat.IoTimeMs,
				stat.IoTimeWeightedMs,
				stat.WriteSectors,
				stat.ReadSectors,
			)
		}
	}

	if err = w.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func collectProcesses(ctx context.Context, c *client.Client) ([]byte, error) {
	resp, err := c.Processes(ctx)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer

	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "PID\tSTATE\tTHREADS\tCPU-TIME\tVIRTMEM\tRESMEM\tCOMMAND")

	for _, msg := range resp.Messages {
		for _, proc := range msg.Processes {
			fmt.Fprintf(w, "%6d\t%1s\t%4d\t%8.2f\t%7s\t%7s\t%s\n",
				proc.Pid,
				proc.State,
				proc.Threads,
				proc.CpuTime,
				humanize.Bytes(proc.VirtualMemory),
				humanize.Bytes(proc.ResidentMemory),
				proc.Command,
			)
		}
	}

	if err = w.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func collectServiceList(ctx context.Context, c *client.Client) ([]string, []byte, error) {
	resp, err := c.ServiceList(ctx)
	if err != nil {
		return nil, nil, err
	}

	serviceIDs := map[string]struct{}{}
	var b strings.Builder

	for _, msg := range resp.Messages {
		for _, svc := range msg.Services {
			serviceIDs[svc.Id] = struct{}{}

			healthy := false
			if svc.GetHealth() != nil {
				healthy = svc.GetHealth().GetHealthy()
			}

			fmt.Fprintf(&b, "id=%s state=%s healthy=%t\n", svc.Id, svc.State, healthy)
		}
	}

	keys := make([]string, 0, len(serviceIDs))
	for id := range serviceIDs {
		keys = append(keys, id)
	}

	slices.Sort(keys)

	return keys, []byte(b.String()), nil
}

func collectServiceState(ctx context.Context, c *client.Client, serviceID string) ([]byte, error) {
	resp, err := c.ServiceInfo(ctx, serviceID)
	if err != nil && resp == nil {
		return nil, fmt.Errorf("error getting service state for %q: %w", serviceID, err)
	}

	var b bytes.Buffer

	if err = formatters.RenderServicesInfo(resp, &b, "", false); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func collectServiceLog(ctx context.Context, c *client.Client, serviceID string) ([]byte, error) {
	stream, err := c.Logs(
		ctx,
		constants.SystemContainerdNamespace,
		commonapi.ContainerDriver_CONTAINERD,
		serviceID,
		false,
		-1,
	)
	if err != nil {
		return nil, err
	}

	return readDataStream(stream.Recv)
}

func readDataStream(recv func() (*commonapi.Data, error)) ([]byte, error) {
	var data []byte

	for {
		resp, err := recv()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		data = append(data, resp.GetBytes()...)
	}

	return data, nil
}

func getDiscoveryConfig() (*clusterresource.Config, error) {
	var config *clusterresource.Config

	if e := WithClient(func(ctx context.Context, c *client.Client) error {
		var err error

		config, err = safe.StateGet[*clusterresource.Config](
			ctx,
			c.COSI,
			resource.NewMetadata(clusterresource.NamespaceName, clusterresource.IdentityType, clusterresource.LocalIdentity, resource.VersionUndefined),
		)

		return err
	}); e != nil {
		return nil, e
	}

	return config, nil
}

func openArchive() (*os.File, error) {
	if supportCmdFlags.output == "" {
		supportCmdFlags.output = "support"

		if config, err := getDiscoveryConfig(); err == nil && config.TypedSpec().DiscoveryEnabled {
			supportCmdFlags.output += "-" + config.TypedSpec().ServiceClusterID
		}

		supportCmdFlags.output += ".zip"
	}

	if _, err := os.Stat(supportCmdFlags.output); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	} else {
		buf := bufio.NewReader(os.Stdin)

		fmt.Printf("%s already exists, overwrite? [y/N]: ", supportCmdFlags.output)

		choice, err := buf.ReadString('\n')
		if err != nil {
			return nil, err
		}

		if strings.TrimSpace(strings.ToLower(choice)) != "y" {
			return nil, fmt.Errorf("operation aborted")
		}
	}

	return os.OpenFile(supportCmdFlags.output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
}

func init() {
	addCommand(supportCmd)
	supportCmd.Flags().StringVarP(&supportCmdFlags.output, "output", "O", "", "output file to write support archive to")
	supportCmd.Flags().IntVarP(&supportCmdFlags.numWorkers, "num-workers", "w", 1, "number of workers per node")
	supportCmd.Flags().BoolVarP(&supportCmdFlags.verbose, "verbose", "v", false, "verbose output")
}
