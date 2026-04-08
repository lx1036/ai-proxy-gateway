package runner

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/lx1036/gateway/pkg/epp/config"
	"github.com/lx1036/gateway/pkg/epp/datalayer/backend/metrics"
	"github.com/lx1036/gateway/pkg/epp/datastore"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/lx1036/gateway/pkg/epp/scheduling"
	"github.com/lx1036/gateway/pkg/epp/server"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	v1 "sigs.k8s.io/gateway-api-inference-extension/api/v1"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1.Install(scheme))
}

type Runner struct {
}

func (r *Runner) Run(ctx context.Context) error {

	opts := server.NewOptions()
	opts.AddFlags(pflag.CommandLine)
	pflag.Parse()
	if err := opts.Complete(); err != nil {
		return err
	}
	if err := opts.Validate(); err != nil {
		klog.Errorf("Failed to validate flags: %v", err)
		return err
	}
	// Print all flag values
	flags := make(map[string]any)
	flag.VisitAll(func(f *flag.Flag) {
		flags[f.Name] = f.Value
	})
	klog.Infof("Flag:\n %+v", flags)

	eppConfig, err := r.parseConfiguration(ctx, opts)

	podMetricsFactory, err := setUpMetrics(opts)

	opt := ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&corev1.Pod{}: {
					Namespaces: map[string]cache.Config{
						gknn.Namespace: {},
					},
				},
				&v1.InferencePool{}: {
					Namespaces: map[string]cache.Config{gknn.Namespace: {FieldSelector: fields.SelectorFromSet(fields.Set{
						"metadata.name": gknn.Name,
					})}},
				},
			},
		},
		Metrics: metricsServerOptions,
	}

	restConfig, err := ctrl.GetConfig()
	if err != nil {
		klog.Errorf("Failed to get Kubernetes rest config: %v", err)
		return err
	}

	mgr, err := ctrl.NewManager(restConfig, opt)
	if err != nil {
		klog.Errorf("Failed to NewManager: %v", err)
		return err
	}

	scheduler := scheduling.NewSchedulerWithConfig(r.schedulerConfig)

	serverRunner := &server.ExtProcRunner{

	}
	if err := serverRunner.SetupWithManager(mgr); err != nil {
		klog.Errorf("Failed to setup EPP controllers: %v", err)
		return err
	}

	if err := mgr.Add(serverRunner.AsRunnable()); err != nil {
		klog.Errorf("Failed to register ext-proc gRPC server runnable: %v", err)
		return err
	}

}

func (r *Runner) parseConfiguration(ctx context.Context, opts *server.Options) {

	configBytes, err := os.ReadFile(opts.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from a file '%s' - %w", opts.ConfigFile, err)
	}

	eppConfig, err := config.LoadConfig(configBytes)

}

func (r *Runner) registerInTreePlugins() {

}

func setUpMetrics(opts *server.Options) (*metrics.PodMetricsFactory, error) {
	mapping, err := metrics.NewMetricMapping(
		opts.TotalQueuedRequestsMetric,
		opts.TotalRunningRequestsMetric,
		opts.KVCacheUsagePercentageMetric,
		opts.LoRAInfoMetric,
		opts.CacheInfoMetric,
	)
	if err != nil {
		klog.Errorf("Failed to create metric mapping from flags: %v", err)
		return nil, err
	}

	var metricsHttpClient *http.Client
	if opts.ModelServerMetricsScheme == "https" {
		metricsHttpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: opts.ModelServerMetricsHTTPSInsecure,
				},
			},
		}
	} else {
		metricsHttpClient = http.DefaultClient
	}

	podMetricsFactory := metrics.NewPodMetricsFactory(&metrics.PodMetricsClient{
		MetricMapping:            mapping,
		ModelServerMetricsPath:   opts.ModelServerMetricsPath,
		ModelServerMetricsScheme: opts.ModelServerMetricsScheme,
		Client:                   metricsHttpClient,
	}, opts.RefreshMetricsInterval)

	return podMetricsFactory, nil
}

func setupDatastore(ctx context.Context, podMetricsFactory *metrics.PodMetricsFactory) {

	datastore.NewDatastore(ctx, podMetricsFactory, modelServerMetricsPort)
}
