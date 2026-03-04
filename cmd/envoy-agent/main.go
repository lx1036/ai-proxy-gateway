package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/lx1036/gateway/cmd/envoy-agent/options"
	"github.com/lx1036/gateway/pkg/agent"
	"github.com/lx1036/gateway/pkg/agent/envoy"
	"github.com/lx1036/gateway/pkg/config"
	"github.com/lx1036/gateway/pkg/util/protomarshal"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"os"
	"os/signal"
	"syscall"
)

var (
	proxyArgs options.ProxyArgs
)

func main() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		klog.Error(err)
		os.Exit(-1)
	}
}

// /usr/local/bin/pilot-agent proxy router --domain higress-system.svc.cluster.local --proxyLogLevel=warning --proxyComponentLogLevel=misc:error --log_output_level=default:info --serviceCluster=higress-gateway

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "ai-proxy-gateway-agent",
		Short:        "ai-proxy-gateway agent.",
		Long:         "ai-proxy-gateway agent runs in the sidecar or gateway container and bootstraps Envoy.",
		SilenceUsage: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
			UnknownFlags: true,
		},
	}

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	proxyCmd := newProxyCommand()
	addFlags(proxyCmd)
	rootCmd.AddCommand(proxyCmd)

	return rootCmd
}

func newProxyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "proxy",
		Short: "XDS proxy agent",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
			UnknownFlags: true,
		},
		//PersistentPreRunE: configureLogging,
		RunE: func(c *cobra.Command, args []string) error {

			proxyConfig, err := config.ConstructProxyConfig(proxyArgs.MeshConfigFile, proxyArgs.ServiceCluster, proxyArgs.Concurrency)
			if err != nil {
				return fmt.Errorf("failed to get proxy config: %v", err)
			}

			if out, err := protomarshal.ToYAML(proxyConfig); err != nil {
				klog.Infof("Failed to serialize to YAML: %v", err)
			} else {
				klog.Infof("Effective config:\n %s", out)
			}

			envoyOptions := envoy.ProxyConfig{
				//LogLevel:           proxyArgs.ProxyLogLevel,
				ComponentLogLevel: proxyArgs.ProxyComponentLogLevel,
				//LogAsJSON:          loggingOptions.JSONEncoding,
				//NodeIPs:            proxyArgs.IPAddresses,
				//Sidecar:            proxyArgs.Type == model.SidecarProxy,
				//OutlierLogPath:     proxyArgs.OutlierLogPath,
				//FileFlushInterval:  proxyConfig.FileFlushInterval,
				//FileFlushMinSizeKB: proxyConfig.FileFlushMinSizeKb,
			}
			agentOptions := options.NewAgentOptions(&proxyArgs, proxyConfig)
			a := agent.NewAgent(proxyConfig, agentOptions, envoyOptions)

			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(errors.New("application shutdown"))
			//defer agent.Close()

			// On SIGINT or SIGTERM, cancel the context, triggering a graceful shutdown
			go WaitSignalFunc(cancel)

			// Start in process SDS, dns server, xds proxy, and Envoy.
			wait, err := a.Run(ctx)
			if err != nil {
				return err
			}
			wait()
			return nil
		},
	}
}

func addFlags(proxyCmd *cobra.Command) {
	proxyArgs = options.NewProxyArgs()
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.DNSDomain, "domain", "",
		"DNS domain suffix. If not provided uses ${POD_NAMESPACE}.svc.cluster.local")
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.MeshConfigFile, "meshConfig", "/etc/istio/config/mesh",
		"File name for Istio mesh configuration. If not specified, a default mesh will be used. This may be overridden by "+
			"PROXY_CONFIG environment variable or proxy.istio.io/config annotation.")
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.ServiceCluster, "serviceCluster", "ai-proxy-gateway", "Service cluster")
	// Log levels are provided by the library https://github.com/gabime/spdlog, used by Envoy.
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.ProxyLogLevel, "proxyLogLevel", "warning",
		fmt.Sprintf("The log level used to start the Envoy proxy (choose from {%s, %s, %s, %s, %s, %s, %s})."+
			"Level may also include one or more scopes, such as 'info,misc:error,upstream:debug'",
			"trace", "debug", "info", "warning", "error", "critical", "off"))
	proxyCmd.PersistentFlags().IntVar(&proxyArgs.Concurrency, "concurrency", 0, "number of worker threads to run")

}

func WaitSignalFunc(cancel context.CancelCauseFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	cancel(fmt.Errorf("received signal: %v", sig.String()))
}
