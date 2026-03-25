package metrics

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/epp/datalayer"
	"github.com/lx1036/gateway/pkg/epp/datalayer/backend"
	"sync/atomic"

	"context"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"go.uber.org/multierr"
	"k8s.io/klog/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
)

const (
	fetchMetricsTimeout = 5 * time.Second

	CacheConfigBlockSizeInfoMetricName = "block_size"
	CacheConfigNumGPUBlocksMetricName  = "num_gpu_blocks"
)

// MetricsState holds the latest state of the latestMetrics that were scraped from a pod.
type MetricsState = datalayer.Metrics

func NewMetricsState() *MetricsState {
	return datalayer.NewMetrics()
}

type PodMetricsClient struct {
	MetricMapping            *MetricMapping
	ModelServerMetricsPath   string
	ModelServerMetricsScheme string

	Client *http.Client
}

func (c *PodMetricsClient) FetchMetrics(ctx context.Context, pod *backend.Pod, existing *MetricsState) (*MetricsState, error) {
	url := c.getMetricEndpoint(pod)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latestMetrics from %s: %w", pod.NamespacedName, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from %s: %v", pod.NamespacedName, resp.StatusCode)
	}

	// text -> latestMetrics
	parser := expfmt.NewTextParser(model.LegacyValidation)
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, err
	}

	return c.promToPodMetrics(metricFamilies, existing)
}

func (c *PodMetricsClient) getMetricEndpoint(pod *backend.Pod) string {
	return c.ModelServerMetricsScheme + "://" + pod.GetMetricsHost() + c.ModelServerMetricsPath
}

func (c *PodMetricsClient) promToPodMetrics(metricFamilies map[string]*dto.MetricFamily, existing *MetricsState) (*MetricsState, error) {
	var errs error
	updated := existing.Clone()

	if c.MetricMapping.TotalQueuedRequests != nil {
		queued, err := c.getMetric(metricFamilies, *c.MetricMapping.TotalQueuedRequests)
		if err == nil {
			updated.WaitingQueueSize = int(queued.GetGauge().GetValue())
		} else {
			errs = multierr.Append(errs, err)
		}
	}

	if c.MetricMapping.TotalRunningRequests != nil {
		running, err := c.getMetric(metricFamilies, *c.MetricMapping.TotalRunningRequests)
		if err == nil {
			updated.RunningRequestsSize = int(running.GetGauge().GetValue())
		} else {
			errs = multierr.Append(errs, err)
		}
	}

	if c.MetricMapping.KVCacheUtilization != nil {
		usage, err := c.getMetric(metricFamilies, *c.MetricMapping.KVCacheUtilization)
		if err == nil {
			updated.KVCacheUsagePercent = usage.GetGauge().GetValue()
		} else {
			errs = multierr.Append(errs, err)
		}
	}

	// vllm:cache_config_info{block_size="16",num_gpu_blocks="1024"} 1
	if c.MetricMapping.CacheConfigInfo != nil {
		cacheMetrics, err := c.getMetric(metricFamilies, *c.MetricMapping.CacheConfigInfo)
		if err != nil {
			errs = multierr.Append(errs, err)
		} else {
			for _, v := range cacheMetrics.GetLabel() {
				switch v.GetName() {
				case CacheConfigBlockSizeInfoMetricName:
					updated.CacheBlockSize, err = strconv.Atoi(v.GetValue())
					if err != nil {
						errs = multierr.Append(errs, err)
					}
				case CacheConfigNumGPUBlocksMetricName:
					updated.CacheNumGPUBlocks, err = strconv.Atoi(v.GetValue())
					if err != nil {
						errs = multierr.Append(errs, err)
					}
				}
			}
		}
	}

	return updated, errs
}

func (c *PodMetricsClient) getMetric(metricFamilies map[string]*dto.MetricFamily, spec MetricSpec) (*dto.Metric, error) {
	metric, ok := metricFamilies[spec.MetricName]
	if !ok {
		return nil, fmt.Errorf("metric family %q not found", spec.MetricName)
	}

	if len(metric.GetMetric()) == 0 {
		return nil, fmt.Errorf("no latestMetrics available for %q", spec.MetricName)
	}

	var latestMetric *dto.Metric
	var latestTimestamp int64 = -1 // Initialize to -1 so any timestamp is greater
	for _, m := range metric.GetMetric() {
		// vllm:num_requests_waiting{model_name="model1"} 0
		if spec.Labels == nil || labelsMatch(m.GetLabel(), spec.Labels) {
			if m.GetTimestampMs() > latestTimestamp {
				latestTimestamp = m.GetTimestampMs()
				latestMetric = m
			}
		}
	}

	if latestMetric != nil {
		return latestMetric, nil
	}

	return nil, fmt.Errorf("no matching metric found for %q with labels %+v", spec.MetricName, spec.Labels)
}

// labelsMatch checks if a metric's labels contain all the labels in the spec.
func labelsMatch(metricLabels []*dto.LabelPair, specLabels map[string]string) bool {
	if len(specLabels) == 0 {
		// 一般 spec 没有 label，只会写 vllm:num_requests_waiting，而不是 vllm:num_requests_waiting{key1=value1,}
		return true // No specific labels required
	}

	for specName, specValue := range specLabels {
		found := false
		for _, label := range metricLabels {
			if label.GetName() == specName && label.GetValue() == specValue {
				found = true
				break
			}
		}
		if !found {
			return false // A required label is missing
		}
	}
	return true // All required labels are present
}

type PodMetricsFactory struct {
	podMetricsClient       *PodMetricsClient
	refreshMetricsInterval time.Duration
}

func NewPodMetricsFactory(podMetricsClient *PodMetricsClient, refreshMetricsInterval time.Duration) *PodMetricsFactory {
	return &PodMetricsFactory{
		podMetricsClient:       podMetricsClient,
		refreshMetricsInterval: refreshMetricsInterval,
	}
}

type PodMetrics struct {
	startOnce sync.Once // ensures the refresh loop goroutine is started only once
	stopOnce  sync.Once

	interval time.Duration
	done     chan struct{}

	podMetricsClient *PodMetricsClient

	metadata atomic.Pointer[datalayer.EndpointMetadata]
	// current latestMetrics
	latestMetrics atomic.Pointer[MetricsState]
}

func (f *PodMetricsFactory) ReleaseEndpoint(podMetrics *PodMetrics) {
	podMetrics.StopRefreshLoop()
}

func (f *PodMetricsFactory) NewEndpoint(parentCtx context.Context, metadata *datalayer.EndpointMetadata) *PodMetrics {
	endpoint := &PodMetrics{
		interval: f.refreshMetricsInterval,
		done:     make(chan struct{}),

		startOnce: sync.Once{},
		stopOnce:  sync.Once{},

		podMetricsClient: f.podMetricsClient,
	}

	endpoint.metadata.Store(metadata)
	endpoint.latestMetrics.Store(NewMetricsState())

	endpoint.startRefreshLoop(parentCtx)
	return endpoint
}

func (pm *PodMetrics) startRefreshLoop(ctx context.Context) {
	pm.startOnce.Do(func() {
		go func() {
			klog.Infof("start to fetch metrics from pod %s/%s for ip: %s/%s",
				pm.GetMetadata().NamespacedName.Namespace, pm.GetMetadata().PodName,
				pm.GetMetadata().Address, pm.GetMetadata().Port)

			ticker := time.NewTicker(pm.interval)
			defer ticker.Stop()
			for {
				select {
				case <-pm.done:
					return
				case <-ctx.Done():
					return
				case <-ticker.C: // refresh latestMetrics periodically
					if err := pm.refreshMetrics(); err != nil {
						klog.Errorf("Failed to refresh latestMetrics: %v", err)
					}
				}
			}
		}()
	})
}

func (pm *PodMetrics) UpdateMetadata(metadata *datalayer.EndpointMetadata) {
	pm.metadata.Store(metadata)
}

func (pm *PodMetrics) GetMetadata() *datalayer.EndpointMetadata {
	return pm.metadata.Load()
}

func (pm *PodMetrics) refreshMetrics() error {
	ctx, cancel := context.WithTimeout(context.Background(), fetchMetricsTimeout)
	defer cancel()
	updated, err := pm.podMetricsClient.FetchMetrics(ctx, pm.GetMetadata(), pm.GetMetrics())
	if err != nil {
		klog.Infof("Failed to refreshed latestMetrics: %v", err)
	}

	// 这里的 updated 是从 existing.Clone() 来的，就算本次循环周期获取失败，updated 中也有上个周期的数据
	// 所以这里的 UpdateMetrics 逻辑没问题
	if updated != nil {
		pm.UpdateMetrics(updated)
	}

	return nil
}

func (pm *PodMetrics) UpdateMetrics(updated *MetricsState) {
	updated.UpdateTime = time.Now()
	klog.Infof("Refreshed latestMetrics updated: %+v", updated)
	pm.latestMetrics.Store(updated)
}

func (pm *PodMetrics) GetMetrics() *MetricsState {
	return pm.latestMetrics.Load()
}

func (pm *PodMetrics) StopRefreshLoop() {
	klog.Infof("stop the metrics loop cycle for pod metadata %+v", pm.GetMetadata())
	pm.stopOnce.Do(func() {
		close(pm.done)
	})
}
