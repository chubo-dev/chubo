//go:build !chubo

package gen

import (
	"github.com/spf13/cobra"

	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func registerKubernetesVersionFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&genConfigCmdFlags.kubernetesVersion, "kubernetes-version", constants.DefaultWorkloadVersion, "desired kubernetes version to run")
}
