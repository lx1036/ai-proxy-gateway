package datastore

import (
	"github.com/lx1036/gateway/pkg/epp/datalayer"
	"github.com/lx1036/gateway/pkg/epp/datalayer/backend/metrics"
	"k8s.io/klog/v2"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

func TestDataStore(test *testing.T) {

	totalQueuedRequestsMetric := "vllm:num_requests_waiting"
	totalRunningRequestsMetric := "vllm:num_requests_running"
	kvCacheUsagePercentageMetric := "vllm:kv_cache_usage_perc"
	loRAInfoMetric := "vllm:lora_requests_info"
	cacheInfoMetric := "vllm:cache_config_info"
	metricMapping, _ := metrics.NewMetricMapping(
		totalQueuedRequestsMetric,
		totalRunningRequestsMetric,
		kvCacheUsagePercentageMetric,
		loRAInfoMetric,
		cacheInfoMetric,
	)

	c := &metrics.PodMetricsClient{
		MetricMapping:            metricMapping,
		ModelServerMetricsPath:   "/metrics",
		ModelServerMetricsScheme: "http",
		Client:                   http.DefaultClient,
	}
	factory := metrics.NewPodMetricsFactory(c, time.Second*10)

	ctx := ctrl.SetupSignalHandler()
	datastore := NewDatastore(ctx, factory)

	cfg := ctrl.GetConfigOrDie()
	// informer watch 并从本地 indexer cache 中获取
	//mgr, _ := ctrl.NewManager(cfg, ctrl.Options{})

	// 每次直接从 apiserver 里取
	k8sClient, err := client.New(cfg, client.Options{})
	if err != nil {
		klog.Fatalf("fail to create k8s client: %v", err)
	}

	//ctrl.NewControllerManagedBy(mgr).
	//	For(&corev1.Pod{}).
	//	WithEventFilter(filter).
	//	Complete(c)

	endpoints := &datalayer.EndpointPool{
		Selector: map[string]string{
			"app.kubernetes.io/name": "llm-d-inference-sim",
		},
		TargetPorts: []int{8000},
		Namespace:   "envoy-gateway-system",
		Name:        "llm-d-inference-sim-inference-pool",
	}

	if err := datastore.StartPodMetricsLoop(ctx, k8sClient, endpoints); err != nil {
		klog.Fatalf("fail to resync infernece pods: %v", err)
	}

	<-ctx.Done()
}
