// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package create

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	clustercmd "github.com/chubo-dev/chubo/cmd/chuboctl/cmd/mgmt/cluster"
	"github.com/chubo-dev/chubo/cmd/chuboctl/cmd/mgmt/cluster/create/clusterops"
	"github.com/chubo-dev/chubo/cmd/chuboctl/cmd/mgmt/cluster/create/clusterops/configmaker"
	clientconfig "github.com/chubo-dev/chubo/pkg/machinery/client/config"
	"github.com/chubo-dev/chubo/pkg/provision/providers"
)

//nolint:gocyclo,cyclop
func createDevCluster(ctx context.Context, cOps clusterops.Common, qOps clusterops.Qemu) error {
	if err := downloadBootAssets(ctx, &qOps); err != nil {
		return err
	}

	if cOps.TalosVersion == "" {
		parts := strings.Split(qOps.NodeInstallImage, ":")
		cOps.TalosVersion = parts[len(parts)-1]
	}

	provisioner, err := providers.Factory(ctx, providers.QemuProviderName)
	if err != nil {
		return err
	}

	clusterConfigs, err := configmaker.GetQemuConfigs(configmaker.QemuOptions{
		ExtraOps:    qOps,
		CommonOps:   cOps,
		Provisioner: provisioner,
	})
	if err != nil {
		return err
	}

	err = preCreate(cOps, clusterConfigs)
	if err != nil {
		return err
	}

	cluster, err := provisioner.Create(ctx, clusterConfigs.ClusterRequest, clusterConfigs.ProvisionOptions...)
	if err != nil {
		return err
	}

	if qOps.DebugShellEnabled {
		fmt.Println("You can now connect to debug shell on any node using these commands:")

		for _, node := range clusterConfigs.ClusterRequest.Nodes {
			chuboDir, err := clientconfig.GetChuboDirectory()
			if err != nil {
				return err
			}

			fmt.Printf("socat - UNIX-CONNECT:%s\n", filepath.Join(chuboDir, "clusters", cOps.RootOps.ClusterName, node.Name+".serial"))
		}

		return nil
	}

	// Create and save the chuboctl configuration file.
	err = postCreate(ctx, cOps, cluster, clusterConfigs)
	if err != nil {
		return err
	}

	return clustercmd.ShowCluster(cluster)
}

func saveConfig(talosConfigObj *clientconfig.Config, talosconfigPath string) (err error) {
	c, err := clientconfig.Open(talosconfigPath)
	if err != nil {
		return fmt.Errorf("error opening node config: %w", err)
	}

	renames := c.Merge(talosConfigObj)
	for _, rename := range renames {
		fmt.Fprintf(os.Stderr, "renamed config context %s\n", rename.String())
	}

	return c.Save(talosconfigPath)
}
