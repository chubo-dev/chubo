//go:build chubo

package gen

import (
	"github.com/spf13/cobra"

	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func registerKubernetesVersionFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&genConfigCmdFlags.kubernetesVersion, "kubernetes-version", constants.DefaultKubernetesVersion, "compatibility alias (ignored in Chubo)")
	cmd.Flags().MarkHidden("kubernetes-version") //nolint:errcheck
}
