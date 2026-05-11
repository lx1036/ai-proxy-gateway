package provider

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/llmrouter/datastore/provider/vllm"
	dto "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type MetricsProvider interface {
	//GetPodMetrics(engine string, pod *corev1.Pod, previousHistogram map[string]*dto.Histogram) (map[string]float64, map[string]*dto.Histogram)
	//GetPodModels(engine string, pod *corev1.Pod) ([]string, error)

	GetPodMetrics(pod *corev1.Pod) (map[string]*dto.MetricFamily, error)
	GetCountMetricsInfo(allMetrics map[string]*dto.MetricFamily) map[string]float64
	GetHistogramPodMetrics(allMetrics map[string]*dto.MetricFamily, previousHistogram map[string]*dto.Histogram) (map[string]float64, map[string]*dto.Histogram)
}

var engineRegistry = map[string]MetricsProvider{
	//"SGLang": sglang.NewMetricsProvider(0),
	"vLLM":   vllm.NewMetricsProvider(0),
}

func GetPodMetrics(engine string, pod *corev1.Pod, previousHistogram map[string]*dto.Histogram) (map[string]float64, map[string]*dto.Histogram) {
	provider, err := GetMetricsProvider(engine)
	if err != nil {
		klog.Errorf("Failed to get inference engine: %v", err)
		return nil, nil
	}

	allMetrics, err := provider.GetPodMetrics(pod)
	if err != nil {
		klog.V(4).Infof("failed to get metrics of pod: %s/%s: %v", pod.GetNamespace(), pod.GetName(), err)
		return nil, nil
	}

	countMetricsInfo := provider.GetCountMetricsInfo(allMetrics)
	histogramMetricsInfo, histogramMetrics := provider.GetHistogramPodMetrics(allMetrics, previousHistogram)

	for name, value := range histogramMetricsInfo {
		// Since the key in countMetricInfo must not be the same as the key in histogramMetricsInfo.
		// You don't have to worry about overriding the value
		countMetricsInfo[name] = value
	}

	return countMetricsInfo, histogramMetrics
}

func GetMetricsProvider(engine string) (MetricsProvider, error) {
	if provider, exists := engineRegistry[engine]; exists {
		return provider, nil
	}

	return nil, fmt.Errorf("unsupported engine: %s", engine)
}
