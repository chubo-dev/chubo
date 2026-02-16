//go:build chubo

package gen

import "github.com/spf13/cobra"

func registerKubernetesPKIFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&genSecretsCmdFlags.fromKubernetesPki, "from-kubernetes-pki", "p", "", "compatibility input for importing an existing PKI directory")
	cmd.Flags().StringVarP(&genSecretsCmdFlags.kubernetesBootstrapToken, "kubernetes-bootstrap-token", "t", "", "compatibility input for importing an existing bootstrap token")
	cmd.Flags().MarkHidden("from-kubernetes-pki")        //nolint:errcheck
	cmd.Flags().MarkHidden("kubernetes-bootstrap-token") //nolint:errcheck
}
