package sglang

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/llmrouter/datastore/provider/metrics"
	dto "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
)

type MetricsProvider struct {
	// The address of vllm's query metrics is http://{model server}:MetricPort/metrics
	// Default is 8000
	MetricPort uint32
}

func NewMetricsProvider(metricPort uint32) *MetricsProvider {
	return &MetricsProvider{
		MetricPort: metricPort,
	}
}

func (provider *MetricsProvider) GetPodMetrics(pod *corev1.Pod) (map[string]*dto.MetricFamily, error) {
	url := fmt.Sprintf("http://%s:%d/metrics", pod.Status.PodIP, provider.MetricPort)
	allMetrics, err := metrics.ParseMetricsURL(url)
	if err != nil {
		return nil, err
	}

	return allMetrics, nil
}
