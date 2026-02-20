//go:build !chubo

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package nodes

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/meta"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/gosuri/uiprogress"
	"github.com/siderolabs/go-talos-support/support"
	"github.com/siderolabs/go-talos-support/support/bundle"
	"github.com/siderolabs/go-talos-support/support/collectors"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/chubo-dev/chubo/pkg/machinery/api/common"
	"github.com/chubo-dev/chubo/pkg/machinery/client"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/formatters"
	clusterresource "github.com/chubo-dev/chubo/pkg/machinery/resources/cluster"
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
	- All Talos internal services logs.
	- Talos COSI resources without secrets.
	- COSI runtime state graph.
	- Processes snapshot.
	- IO pressure snapshot.
	- Mounts list.
	- PCI devices info.
	- Talos version.
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

		progress := make(chan bundle.Progress)

		var (
			eg     errgroup.Group
			errors supportBundleErrors
		)

		eg.Go(func() error {
			if supportCmdFlags.verbose {
				for p := range progress {
					errors.handleProgress(p)
				}
			} else {
				showProgress(progress, &errors)
			}

			return nil
		})

		collectErr := collectData(f, progress)

		close(progress)

		if e := eg.Wait(); e != nil {
			return e
		}

		if err = errors.print(); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Support bundle is written to %s\n", supportCmdFlags.output)

		return collectErr
	},
}

func collectData(dest *os.File, progress chan bundle.Progress) error {
	return WithClientNoNodes(func(ctx context.Context, c *client.Client) error {
		opts := []bundle.Option{
			bundle.WithArchiveOutput(dest),
			bundle.WithTalosClient(c),
			bundle.WithNodes(GlobalArgs.Nodes...),
			bundle.WithNumWorkers(supportCmdFlags.numWorkers),
			bundle.WithProgressChan(progress),
		}

		if !supportCmdFlags.verbose {
			opts = append(opts, bundle.WithLogOutput(io.Discard))
		}

		options := bundle.NewOptions(opts...)

		supportCollectors, err := getTalosOnlyCollectors(ctx, c)
		if err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Talos-only collector mode active.")

		return support.CreateSupportBundle(ctx, options, supportCollectors...)
	})
}

func getTalosOnlyCollectors(ctx context.Context, c *client.Client) ([]*collectors.Collector, error) {
	var allCollectors []*collectors.Collector

	for _, node := range GlobalArgs.Nodes {
		nodeCollectors, err := getTalosOnlyNodeCollectors(client.WithNode(ctx, node), c)
		if err != nil {
			return nil, err
		}

		allCollectors = append(allCollectors, collectors.WithNode(nodeCollectors, node)...)
	}

	return allCollectors, nil
}

func getTalosOnlyNodeCollectors(ctx context.Context, c *client.Client) ([]*collectors.Collector, error) {
	supportCollectors := []*collectors.Collector{
		collectors.NewCollector("summary", talosOnlySummary),
		collectors.NewCollector("dmesg.log", talosOnlyDmesg),
		collectors.NewCollector("controller-runtime.log", talosOnlyServiceLog("controller-runtime")),
		collectors.NewCollector("dns-resolve-cache.log", talosOnlyServiceLog("dns-resolve-cache")),
		collectors.NewCollector("dependencies.dot", talosOnlyDependencies),
		collectors.NewCollector("mounts", talosOnlyMounts),
		collectors.NewCollector("devices", talosOnlyDevices),
		collectors.NewCollector("io", talosOnlyIOPressure),
		collectors.NewCollector("processes", talosOnlyProcesses),
		collectors.NewCollector("service-list.log", talosOnlyServiceList),
	}

	resourceCollectors, err := getTalosOnlyResourceCollectors(ctx, c.COSI)
	if err != nil {
		supportCollectors = append(
			supportCollectors,
			collectors.WithFolder([]*collectors.Collector{
				collectors.NewCollector("list-error.log", staticCollector(
					fmt.Sprintf("failed to list COSI resource definitions: %s\n", err),
				)),
			}, "resources")...,
		)
	} else {
		supportCollectors = append(
			supportCollectors,
			collectors.WithFolder(resourceCollectors, "resources")...,
		)
	}

	serviceCollectors, err := getTalosOnlyServiceCollectors(ctx, c)
	if err != nil {
		supportCollectors = append(
			supportCollectors,
			collectors.NewCollector("service-list-error.log", staticCollector(
				fmt.Sprintf("failed to list service collectors: %s\n", err),
			)),
		)

		return supportCollectors, nil
	}

	supportCollectors = append(
		supportCollectors,
		collectors.WithFolder(serviceCollectors, "service-logs")...,
	)

	return supportCollectors, nil
}

func getTalosOnlyServiceCollectors(ctx context.Context, c *client.Client) ([]*collectors.Collector, error) {
	resp, err := c.ServiceList(ctx)
	if err != nil {
		return nil, err
	}

	var collectorsList []*collectors.Collector

	serviceIDs := map[string]struct{}{}

	for _, msg := range resp.Messages {
		for _, svc := range msg.Services {
			serviceIDs[svc.Id] = struct{}{}
		}
	}

	keys := make([]string, 0, len(serviceIDs))
	for id := range serviceIDs {
		keys = append(keys, id)
	}

	slices.Sort(keys)

	for _, id := range keys {
		serviceID := id

		collectorsList = append(collectorsList,
			collectors.NewCollector(fmt.Sprintf("%s.log", serviceID), talosOnlyServiceLog(serviceID)),
			collectors.NewCollector(fmt.Sprintf("%s.state", serviceID), talosOnlyServiceInfo(serviceID)),
		)
	}

	return collectorsList, nil
}

func getTalosOnlyResourceCollectors(ctx context.Context, cosiState state.State) ([]*collectors.Collector, error) {
	resourceDefinitions, err := safe.StateListAll[*meta.ResourceDefinition](ctx, cosiState)
	if err != nil {
		return nil, err
	}

	var collectorsList []*collectors.Collector

	resourceDefinitions.ForEach(func(rd *meta.ResourceDefinition) {
		collectorsList = append(
			collectorsList,
			collectors.NewCollector(
				fmt.Sprintf("%s.yaml", rd.Metadata().ID()),
				talosOnlyResource(rd),
			),
		)
	})

	return collectorsList, nil
}

func talosOnlySummary(ctx context.Context, options *bundle.Options) ([]byte, error) {
	resp, err := options.TalosClient.Version(ctx)
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

func talosOnlyDmesg(ctx context.Context, options *bundle.Options) ([]byte, error) {
	stream, err := options.TalosClient.Dmesg(ctx, false, false)
	if err != nil {
		return nil, err
	}

	var data []byte

	for {
		resp, err := stream.Recv()
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

func talosOnlyDependencies(ctx context.Context, options *bundle.Options) ([]byte, error) {
	resp, err := options.TalosClient.Inspect.ControllerRuntimeDependencies(ctx)
	if err != nil && resp == nil {
		return nil, fmt.Errorf("error getting controller runtime dependencies: %w", err)
	}

	var b bytes.Buffer

	if err := formatters.RenderGraph(ctx, options.TalosClient, resp, &b, true); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func talosOnlyMounts(ctx context.Context, options *bundle.Options) ([]byte, error) {
	resp, err := options.TalosClient.Mounts(ctx)
	if err != nil && resp == nil {
		return nil, fmt.Errorf("error getting mounts: %w", err)
	}

	var b bytes.Buffer

	if err := formatters.RenderMounts(resp, &b, nil); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func talosOnlyDevices(ctx context.Context, options *bundle.Options) ([]byte, error) {
	reader, err := options.TalosClient.Read(ctx, "/proc/bus/pci/devices")
	if err != nil {
		return nil, err
	}

	defer reader.Close() //nolint:errcheck

	return io.ReadAll(reader)
}

func talosOnlyIOPressure(ctx context.Context, options *bundle.Options) ([]byte, error) {
	resp, err := options.TalosClient.MachineClient.DiskStats(ctx, &emptypb.Empty{})
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

	if err := w.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func talosOnlyProcesses(ctx context.Context, options *bundle.Options) ([]byte, error) {
	resp, err := options.TalosClient.Processes(ctx)
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

	if err := w.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func talosOnlyServiceList(ctx context.Context, options *bundle.Options) ([]byte, error) {
	resp, err := options.TalosClient.ServiceList(ctx)
	if err != nil {
		return nil, err
	}

	var b strings.Builder

	for _, msg := range resp.Messages {
		for _, svc := range msg.Services {
			healthy := false
			if svc.GetHealth() != nil {
				healthy = svc.GetHealth().GetHealthy()
			}

			fmt.Fprintf(&b, "id=%s state=%s healthy=%t\n", svc.Id, svc.State, healthy)
		}
	}

	return []byte(b.String()), nil
}

func talosOnlyServiceInfo(serviceID string) collectors.Collect {
	return func(ctx context.Context, options *bundle.Options) ([]byte, error) {
		resp, err := options.TalosClient.ServiceInfo(ctx, serviceID)
		if err != nil && resp == nil {
			return nil, fmt.Errorf("error getting service state for %q: %w", serviceID, err)
		}

		var b bytes.Buffer

		if err := formatters.RenderServicesInfo(resp, &b, "", false); err != nil {
			return nil, err
		}

		return b.Bytes(), nil
	}
}

func talosOnlyServiceLog(serviceID string) collectors.Collect {
	return func(ctx context.Context, options *bundle.Options) ([]byte, error) {
		stream, err := options.TalosClient.Logs(
			ctx,
			constants.SystemContainerdNamespace,
			common.ContainerDriver_CONTAINERD,
			serviceID,
			false,
			-1,
		)
		if err != nil {
			return nil, err
		}

		var data []byte

		for {
			resp, err := stream.Recv()
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
}

func talosOnlyResource(rd *meta.ResourceDefinition) collectors.Collect {
	return func(ctx context.Context, options *bundle.Options) ([]byte, error) {
		resp, err := options.TalosClient.COSI.List(
			ctx,
			resource.NewMetadata(
				rd.TypedSpec().DefaultNamespace,
				rd.TypedSpec().Type,
				"",
				resource.VersionUndefined,
			),
		)
		if err != nil {
			return nil, err
		}

		var (
			b        bytes.Buffer
			hasItems bool
		)

		encoder := yaml.NewEncoder(&b)

		for _, item := range resp.Items {
			encoded := struct {
				Metadata *resource.Metadata `yaml:"metadata"`
				Spec     interface{}        `yaml:"spec"`
			}{
				Metadata: item.Metadata(),
				Spec:     "<REDACTED>",
			}

			if rd.TypedSpec().Sensitivity != meta.Sensitive {
				encoded.Spec = item.Spec()
			}

			if err := encoder.Encode(&encoded); err != nil {
				return nil, err
			}

			hasItems = true
		}

		if !hasItems {
			return nil, nil
		}

		if err := encoder.Close(); err != nil {
			return nil, err
		}

		return b.Bytes(), nil
	}
}

func staticCollector(data string) collectors.Collect {
	return func(_ context.Context, _ *bundle.Options) ([]byte, error) {
		return []byte(data), nil
	}
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

type supportBundleError struct {
	source string
	value  string
}

type supportBundleErrors struct {
	errors []supportBundleError
}

func (sbe *supportBundleErrors) handleProgress(p bundle.Progress) {
	if p.Error != nil {
		sbe.errors = append(sbe.errors, supportBundleError{
			source: p.Source,
			value:  p.Error.Error(),
		})
	}
}

func (sbe *supportBundleErrors) print() error {
	if sbe.errors == nil {
		return nil
	}

	var wroteHeader bool

	w := tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)

	for _, err := range sbe.errors {
		if !wroteHeader {
			wroteHeader = true

			fmt.Fprintln(os.Stderr, "Processed with errors:")
			fmt.Fprintln(w, "\tSOURCE\tERROR")
		}

		details := strings.Split(err.value, "\n")
		for i, d := range details {
			details[i] = strings.TrimSpace(d)
		}

		fmt.Fprintf(w, "\t%s\t%s\n", err.source, color.RedString(details[0]))

		if len(details) > 1 {
			for _, line := range details[1:] {
				fmt.Fprintf(w, "\t\t%s\n", color.RedString(line))
			}
		}
	}

	return w.Flush()
}

func showProgress(progress <-chan bundle.Progress, errors *supportBundleErrors) {
	uiprogress.Start()

	type nodeProgress struct {
		mu    sync.Mutex
		state string
		bar   *uiprogress.Bar
	}

	nodes := map[string]*nodeProgress{}

	for p := range progress {
		errors.handleProgress(p)

		var (
			np *nodeProgress
			ok bool
		)

		src := p.Source

		if _, ok = nodes[p.Source]; !ok {
			bar := uiprogress.AddBar(p.Total)
			bar = bar.AppendCompleted().PrependElapsed()

			np = &nodeProgress{
				state: "initializing...",
				bar:   bar,
			}

			bar.AppendFunc(
				func(src string, np *nodeProgress) func(b *uiprogress.Bar) string {
					return func(b *uiprogress.Bar) string {
						np.mu.Lock()
						defer np.mu.Unlock()

						return fmt.Sprintf("%s: %s", src, np.state)
					}
				}(src, np),
			)

			bar.Width = 20

			nodes[src] = np
		} else {
			np = nodes[src]
		}

		np.mu.Lock()
		np.state = p.State
		np.mu.Unlock()

		np.bar.Incr()
	}

	uiprogress.Stop()
}

func init() {
	addCommand(supportCmd)
	supportCmd.Flags().StringVarP(&supportCmdFlags.output, "output", "O", "", "output file to write support archive to")
	supportCmd.Flags().IntVarP(&supportCmdFlags.numWorkers, "num-workers", "w", 1, "number of workers per node")
	supportCmd.Flags().BoolVarP(&supportCmdFlags.verbose, "verbose", "v", false, "verbose output")
}
