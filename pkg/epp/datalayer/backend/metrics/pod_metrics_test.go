package metrics

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/epp/datalayer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"
)

func TestPodMetrics(test *testing.T) {
	totalQueuedRequestsMetric := "vllm:num_requests_waiting"
	totalRunningRequestsMetric := "vllm:num_requests_running"
	kvCacheUsagePercentageMetric := "vllm:kv_cache_usage_perc"
	loRAInfoMetric := "vllm:lora_requests_info"
	cacheInfoMetric := "vllm:cache_config_info"
	metricMapping, _ := NewMetricMapping(
		totalQueuedRequestsMetric,
		totalRunningRequestsMetric,
		kvCacheUsagePercentageMetric,
		loRAInfoMetric,
		cacheInfoMetric,
	)

	client := &PodMetricsClient{
		MetricMapping:            metricMapping,
		ModelServerMetricsPath:   "/metrics",
		ModelServerMetricsScheme: "http",
		Client:                   http.DefaultClient,
	}
	factory := NewPodMetricsFactory(client, time.Second*10)

	podName := "vllm-simulator-llm-d-inference-sim-5c59675f54-j8sjx"
	// k port-forward -n envoy-gateway-system pod/vllm-simulator-llm-d-inference-sim-5c59675f54-j8sjx 8000:8000
	podIP := "127.0.0.1"
	metricsPort := "8000"
	metadata := &datalayer.EndpointMetadata{
		NamespacedName: types.NamespacedName{
			Name:      fmt.Sprintf("%s-rank-0", podName),
			Namespace: "envoy-gateway-system",
		},
		PodName:     podName,
		Address:     podIP,
		Port:        metricsPort,
		MetricsHost: net.JoinHostPort(podIP, metricsPort),
		Labels:      nil,
	}

	ctx := ctrl.SetupSignalHandler()
	factory.NewEndpoint(ctx, metadata)
	<-ctx.Done()
}

func TestMap(test *testing.T) {
	labels := map[string]string{
		"app": "vllm-simulator-llm-d-inference-sim",
	}

	type EndpointMetadata struct {
		labels map[string]string
	}
	metadata := &EndpointMetadata{
		labels: labels,
	}
	metadata.labels["app"] = "test"
	// 会修改 labels 原数据, 需要重新复制

	klog.Infof("metadata: %v", metadata)
	klog.Infof("labels: %v", labels)
	// I0325 17:44:53.867191   36157 pod_metrics_test.go:71] metadata: &{map[app:test]}
	//I0325 17:44:53.867326   36157 pod_metrics_test.go:72] labels: map[app:test]
}
