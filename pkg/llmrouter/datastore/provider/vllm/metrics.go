package vllm

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/llmrouter/datastore/provider/metrics"
	dto "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
)

const (
	GPUCacheUsage     = "vllm:gpu_cache_usage_perc"
	RequestWaitingNum = "vllm:num_requests_waiting"
	RequestRunningNum = "vllm:num_requests_running"
	TPOT              = "vllm:time_per_output_token_seconds"
	TTFT              = "vllm:time_to_first_token_seconds"

	Port = 8000
)

var (
	CounterAndGaugeMetrics = []string{
		GPUCacheUsage,
		RequestWaitingNum,
		RequestRunningNum,
	}

	HistogramMetrics = []string{
		TPOT,
		TTFT,
	}
)

type MetricsProvider struct {
	// The address of vllm's query metrics is http://{model server}:MetricPort/metrics
	// Default is 8000
	MetricPort uint32
}

func NewMetricsProvider(metricPort uint32) *MetricsProvider {
	if metricPort == 0 {
		metricPort = Port
	}

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

func (provider *MetricsProvider) GetCountMetricsInfo(allMetrics map[string]*dto.MetricFamily) map[string]float64 {
	wantMetrics := make(map[string]float64)
	for _, metricName := range CounterAndGaugeMetrics {
		metricInfo, exist := allMetrics[metricName]
		if !exist {
			continue
		}
		for _, metric := range metricInfo.Metric {
			metricValue := metric.GetGauge().GetValue()
			wantMetrics[mapOfMetricsName[metricName]] = metricValue
		}
	}

	return wantMetrics
}

func (provider *MetricsProvider) GetHistogramPodMetrics(allMetrics map[string]*dto.MetricFamily, previousHistogram map[string]*dto.Histogram) (map[string]float64, map[string]*dto.Histogram) {
	wantMetrics := make(map[string]float64)
	histogramMetrics := make(map[string]*dto.Histogram)
	for _, metricName := range HistogramMetrics {
		metricInfo, exist := allMetrics[metricName]
		if !exist {
			continue
		}
		for _, metric := range metricInfo.Metric {
			histogram := metric.GetHistogram()
			histogramMetrics[mapOfMetricsName[metricName]] = histogram
			previousMetric := previousHistogram[mapOfMetricsName[metricName]]
			if previousMetric == nil {
				// Ignore the effects of history and give each pod a fair chance at the initial.
				wantMetrics[mapOfMetricsName[metricName]] = float64(0.0)
			} else {
				wantMetrics[mapOfMetricsName[metricName]] = metrics.LastPeriodAvg(previousMetric, histogram)
			}
		}
	}

	return wantMetrics, histogramMetrics
}
