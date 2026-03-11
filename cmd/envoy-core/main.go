package main

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/core/bootstrap"
	"github.com/spf13/cobra"
	"istio.io/istio/pkg/cmd"
	"istio.io/istio/pkg/config/constants"
	"os"

	"istio.io/istio/pkg/keepalive"
	"istio.io/istio/pkg/log"
)

var (
	serverArgs *bootstrap.ServerArgs

	loggingOptions = log.DefaultOptions()

	waitForMonitorSignal = func(stop chan struct{}) {
		cmd.WaitSignal(stop)
	}
)

func main() {
	if err := GetRootCommand().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func GetRootCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "envoy-core",
		Short: "envoy-core",
		Long:  "Next-generation Cloud Native Gateway",
	}

	command.AddCommand(getServerCommand())

	return command
}

func getServerCommand() *cobra.Command {
	serveCmd := &cobra.Command{
		Use:     "serve",
		Aliases: []string{"serve"},
		Short:   "Starts the higress ingress controller",
		Example: "higress serve",
		PreRunE: func(c *cobra.Command, args []string) error {
			return log.Configure(loggingOptions)
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd.PrintFlags(c.Flags())

			stop := make(chan struct{})

			server, err := bootstrap.NewServer(serverArgs)
			if err != nil {
				return fmt.Errorf("fail to create higress server: %v", err)
			}

			if err := server.Start(stop); err != nil {
				return fmt.Errorf("fail to start higress server: %v", err)
			}

			waitForMonitorSignal(stop)

			server.WaitUntilCompletion()
			return nil
		},
	}

	serverArgs = &bootstrap.ServerArgs{
		//Debug:                true,
		//NativeIstio:          true,
		//HttpAddress:          ":8888",
		//CertHttpAddress:      ":8889",
		GrpcAddress:          ":15051",
		GrpcKeepAliveOptions: keepalive.DefaultOption(),
		//XdsOptions: bootstrap.XdsOptions{
		//	DebounceAfter:         features.DebounceAfter,
		//	DebounceMax:           features.DebounceMax,
		//	EnableEDSDebounce:     features.EnableEDSDebounce,
		//	KeepConfigLabels:      keepConfigLabels,
		//	KeepConfigAnnotations: keepConfigAnnotations,
		//},
	}

	serveCmd.PersistentFlags().StringVar(&serverArgs.GatewaySelectorKey, "gatewaySelectorKey", "higress", "gateway resource selector label key")
	serveCmd.PersistentFlags().StringVar(&serverArgs.GatewaySelectorValue, "gatewaySelectorValue", "higress-system-higress-gateway", "gateway resource selector label value")
	//serveCmd.PersistentFlags().BoolVar(&serverArgs.EnableStatus, "enableStatus", true, "enable the ingress status syncer which use to update the ip in ingress's status")
	//serveCmd.PersistentFlags().StringVar(&serverArgs.IngressClass, "ingressClass", innerconstants.DefaultIngressClass, "if not empty, only watch the ingresses have the specified class, otherwise watch all ingresses")
	//serveCmd.PersistentFlags().StringVar(&serverArgs.WatchNamespace, "watchNamespace", "", "if not empty, only wath the ingresses in the specified namespace, otherwise watch in all namespacees")
	//

	//serveCmd.PersistentFlags().BoolVar(&serverArgs.Debug, "debug", serverArgs.Debug, "if true, enables more debug http api")
	//serveCmd.PersistentFlags().StringVar(&serverArgs.HttpAddress, "httpAddress", serverArgs.HttpAddress, "the http address")
	serveCmd.PersistentFlags().StringVar(&serverArgs.GrpcAddress, "grpcAddress", serverArgs.GrpcAddress, "the grpc address")
	//serveCmd.PersistentFlags().BoolVar(&serverArgs.KeepStaleWhenEmpty, "keepStaleWhenEmpty", false, "keep the stale service entry when there are no endpoints in the service")
	serveCmd.PersistentFlags().StringVar(&serverArgs.RegistryOptions.ClusterRegistriesNamespace, "clusterRegistriesNamespace",
		serverArgs.RegistryOptions.ClusterRegistriesNamespace, "Namespace for ConfigMap which stores clusters configs")
	serveCmd.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeConfig, "kubeconfig", "",
		"Use a Kubernetes configuration file instead of in-cluster configuration")
	// RegistryOptions Controller options
	serveCmd.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeOptions.DomainSuffix, "domain", constants.DefaultClusterLocalDomain,
		"DNS domain suffix")
	serveCmd.PersistentFlags().StringVar((*string)(&serverArgs.RegistryOptions.KubeOptions.ClusterID), "clusterID", "Kubernetes",
		"The ID of the cluster that this instance resides")
	serveCmd.PersistentFlags().StringToStringVar(&serverArgs.RegistryOptions.KubeOptions.ClusterAliases, "clusterAliases", map[string]string{},
		"Alias names for clusters")

	serveCmd.PersistentFlags().Float32Var(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIQPS, "kubernetesApiQPS", 80.0,
		"Maximum QPS when communicating with the kubernetes API")
	serveCmd.PersistentFlags().IntVar(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIBurst, "kubernetesApiBurst", 160,
		"Maximum burst for throttle when communicating with the kubernetes API")

	//serveCmd.PersistentFlags().Uint32Var(&serverArgs.GatewayHttpPort, "gatewayHttpPort", 80,
	//	"Http listening port of gateway pod")
	//serveCmd.PersistentFlags().Uint32Var(&serverArgs.GatewayHttpsPort, "gatewayHttpsPort", 443,
	//	"Https listening port of gateway pod")
	//
	//serveCmd.PersistentFlags().BoolVar(&serverArgs.EnableAutomaticHttps, "enableAutomaticHttps", false, "if true, enables automatic https")
	//serveCmd.PersistentFlags().StringVar(&serverArgs.AutomaticHttpsEmail, "automaticHttpsEmail", "", "email for automatic https")
	//serveCmd.PersistentFlags().StringVar(&serverArgs.CertHttpAddress, "certHttpAddress", serverArgs.CertHttpAddress, "the cert http address")

	loggingOptions.AttachCobraFlags(serveCmd)
	serverArgs.GrpcKeepAliveOptions.AttachCobraFlags(serveCmd)

	return serveCmd
}
