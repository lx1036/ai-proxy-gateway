package server

import (
	"github.com/spf13/pflag"
	"time"
)

const (
	DefaultGrpcPort = 9002
)

type Options struct {
	GRPCPort int

	ConfigFile string // The path to the configuration file.

	ModelServerMetricsScheme        string        // Protocol scheme used in scraping metrics from endpoints.
	ModelServerMetricsPath          string        // URL path used in scraping metrics from endpoints.
	ModelServerMetricsHTTPSInsecure bool          // Disable certificate verification when using 'https' scheme for 'model-server-metrics-scheme'.
	TotalQueuedRequestsMetric       string        // Prometheus metric specification for the number of queued requests.
	TotalRunningRequestsMetric      string        // Prometheus metric specification for the number of running requests.
	KVCacheUsagePercentageMetric    string        // Prometheus metric specification for the fraction of KV-cache blocks currently in use.
	LoRAInfoMetric                  string        // Prometheus metric specification for the LoRA info metrics.
	CacheInfoMetric                 string        // Prometheus metric specification for the cache info metrics.
	RefreshMetricsInterval          time.Duration // Interval to refresh metrics.

	// internal
	fs *pflag.FlagSet // FlagSet used in AddFlags() and consulted in Complete()
}

func NewOptions() *Options {
	return &Options{
		GRPCPort: DefaultGrpcPort,

		ModelServerMetricsScheme:     "http",
		ModelServerMetricsPath:       "/metrics",
		TotalQueuedRequestsMetric:    "vllm:num_requests_waiting",
		TotalRunningRequestsMetric:   "vllm:num_requests_running",
		KVCacheUsagePercentageMetric: "vllm:kv_cache_usage_perc",
		LoRAInfoMetric:               "vllm:lora_requests_info",
		CacheInfoMetric:              "vllm:cache_config_info",
		RefreshMetricsInterval:       1 * time.Second,
		//RefreshMetricsInterval:           500 * time.Millisecond,

	}
}

func (opts *Options) AddFlags(fs *pflag.FlagSet) {
	opts.fs = fs

	fs.IntVar(&opts.GRPCPort, "grpc-port", opts.GRPCPort, "gRPC port used for communicating with Envoy proxy.")
	fs.StringVar(&opts.ConfigFile, "config-file", opts.ConfigFile, "The path to the configuration file.")

	fs.StringVar(&opts.ModelServerMetricsScheme, "model-server-metrics-scheme", opts.ModelServerMetricsScheme,
		"Protocol scheme used in scraping metrics from endpoints.")
}
