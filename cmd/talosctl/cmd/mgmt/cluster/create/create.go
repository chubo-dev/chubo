// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package create

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-getter/v2"
	"github.com/siderolabs/go-retry/retry"

	"github.com/chubo-dev/chubo/cmd/talosctl/cmd/mgmt/cluster/create/clusterops"
	clientconfig "github.com/chubo-dev/chubo/pkg/machinery/client/config"
	"github.com/chubo-dev/chubo/pkg/provision"
	"github.com/chubo-dev/chubo/pkg/provision/access"
)

// downloadBootAssets downloads the boot assets in the given qemuOps if they are URLs, and replaces their URL paths with the downloaded paths on the filesystem.
//
// As it modifies the qemuOps struct, it needs to be passed by reference.
//
//nolint:gocyclo
func downloadBootAssets(ctx context.Context, qOps *clusterops.Qemu) error {
	// download & cache images if provides as URLs
	for _, downloadableImage := range []struct {
		path           *string
		disableArchive bool
	}{
		{
			path: &qOps.NodeVmlinuzPath,
		},
		{
			path:           &qOps.NodeInitramfsPath,
			disableArchive: true,
		},
		{
			path: &qOps.NodeISOPath,
		},
		{
			path: &qOps.NodeUSBPath,
		},
		{
			path: &qOps.NodeUKIPath,
		},
		{
			path: &qOps.NodeDiskImagePath,
			// we disable extracting the compressed image since we handle zstd for disk images
			disableArchive: true,
		},
	} {
		if *downloadableImage.path == "" {
			continue
		}

		u, err := url.Parse(*downloadableImage.path)
		if err != nil || !(u.Scheme == "http" || u.Scheme == "https") {
			// not a URL
			continue
		}

		defaultStateDir, err := clientconfig.GetTalosDirectory()
		if err != nil {
			return err
		}

		cacheDir := filepath.Join(defaultStateDir, "cache")

		if err = os.MkdirAll(cacheDir, 0o755); err != nil {
			return err
		}

		destPath := strings.ReplaceAll(
			strings.ReplaceAll(u.String(), "/", "-"),
			":", "-")

		_, err = os.Stat(filepath.Join(cacheDir, destPath))
		if err == nil {
			*downloadableImage.path = filepath.Join(cacheDir, destPath)

			// already cached
			continue
		}

		fmt.Fprintf(os.Stderr, "downloading asset from %q to %q\n", u.String(), filepath.Join(cacheDir, destPath))

		client := getter.Client{
			Getters: []getter.Getter{
				&getter.HttpGetter{
					HeadFirstTimeout: 30 * time.Minute,
					ReadTimeout:      30 * time.Minute,
				},
			},
		}

		if downloadableImage.disableArchive {
			q := u.Query()

			q.Set("archive", "false")

			u.RawQuery = q.Encode()
		}

		_, err = client.Get(ctx, &getter.Request{
			Src:     u.String(),
			Dst:     filepath.Join(cacheDir, destPath),
			GetMode: getter.ModeFile,
		})
		if err != nil {
			// clean up the destination on failure
			os.Remove(filepath.Join(cacheDir, destPath)) //nolint:errcheck

			return err
		}

		*downloadableImage.path = filepath.Join(cacheDir, destPath)
	}

	return nil
}

func postCreate(
	ctx context.Context,
	cOps clusterops.Common,
	cluster provision.Cluster,
	clusterConfigs clusterops.ClusterConfigs,
) error {
	if clusterConfigs.ConfigBundle != nil {
		bundleTalosconfig := clusterConfigs.ConfigBundle.TalosConfig()

		if err := saveConfig(bundleTalosconfig, cOps.TalosconfigDestination); err != nil {
			return err
		}
	}

	clusterAccess := access.NewAdapter(cluster, clusterConfigs.ProvisionOptions...)
	defer clusterAccess.Close() //nolint:errcheck

	if cOps.ApplyConfigEnabled {
		fmt.Println("applying configuration to the cluster nodes")

		err := clusterAccess.ApplyConfig(ctx, clusterConfigs.ClusterRequest.Nodes, clusterConfigs.ClusterRequest.SiderolinkRequest, os.Stdout)
		if err != nil {
			return err
		}
	}

	if cOps.OmniAPIEndpoint != "" || (cOps.SkipInjectingConfig && !cOps.ApplyConfigEnabled) {
		return nil
	}

	// In chubo, `cluster create` is a dev fixture creator only:
	// it should not bootstrap legacy control-plane services or attempt legacy client-config export/merge.
	if cOps.ClusterWait {
		if err := waitForRuntimeAPI(ctx, clusterAccess, clusterConfigs.ClusterRequest.Nodes, cOps.ClusterWaitTimeout); err != nil {
			return err
		}
	}

	return nil
}

func waitForRuntimeAPI(ctx context.Context, clusterAccess *access.Adapter, nodes []provision.NodeRequest, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("invalid wait timeout: %s", timeout)
	}

	deadline := time.Now().Add(timeout)

	for _, node := range nodes {
		ep := node.IPs[0].String()

		fmt.Fprintf(os.Stderr, "waiting for runtime OS API on %s (%s)\n", node.Name, ep)

		waitNode := func(ctx context.Context) error {
			cli, err := clusterAccess.Client(ep)
			if err != nil {
				return retry.ExpectedError(err)
			}

			attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			_, err = cli.Version(attemptCtx)
			if err != nil {
				return retry.ExpectedError(err)
			}

			return nil
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("timed out waiting for runtime OS API on %s (%s)", node.Name, ep)
		}

		if err := retry.Constant(remaining, retry.WithUnits(500*time.Millisecond), retry.WithJitter(100*time.Millisecond)).RetryWithContext(ctx, waitNode); err != nil {
			return fmt.Errorf("timed out waiting for runtime OS API on %s (%s): %w", node.Name, ep, err)
		}
	}

	return nil
}
