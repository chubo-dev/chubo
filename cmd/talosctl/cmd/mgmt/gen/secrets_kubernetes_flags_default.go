//go:build !chubo

package gen

import "github.com/spf13/cobra"

func registerKubernetesPKIFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&genSecretsCmdFlags.fromKubernetesPki, "from-kubernetes-pki", "p", "", "use a Kubernetes PKI directory (e.g. /etc/kubernetes/pki) as input")
	cmd.Flags().StringVarP(&genSecretsCmdFlags.kubernetesBootstrapToken, "kubernetes-bootstrap-token", "t", "", "use the provided bootstrap token as input")
}
