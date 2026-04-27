package envoy_gateway

import (
	"github.com/lx1036/gateway/pkg/envoygateway/version"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main()  {

	cmd := &cobra.Command{
		Use:                        "envoy-gateway",
		Short: "Envoy Gateway",
		Long:  "Manages Envoy Proxy as a standalone or Kubernetes-based application gateway",
	}
	cmd.AddCommand(NewServerCommand())
	cmd.AddCommand(NewVersionCommand())

	if err := cmd.ExecuteContext(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("cmd error: %v", err)
	}
}



func NewVersionCommand() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"versions", "v"},
		Short:   "Show versions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return version.Print(cmd.OutOrStdout(), output)
		},
	}

	cmd.PersistentFlags().StringVarP(&output, "output", "o", "", "One of 'yaml' or 'json'")

	return cmd
}
