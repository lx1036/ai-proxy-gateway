package datastore

import (
	"github.com/lx1036/gateway/pkg/llmrouter/datastore/provider"
	dto "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sync"
	"time"
	"context"
)

type Store interface {
}

type Option func(*store)

type store struct {
	pods sync.Map // map[types.NamespacedName]*PodInfo

}

func New(opts ...Option) Store {
	s := &store{}

	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	return s
}

func (s *store) Run(ctx context.Context) {
	//s.rootCtx = ctx
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				s.pods.Range(func(key, value any) bool {
					if p, ok := value.(*PodInfo); ok {
						s.updatePodMetrics(p)
						s.updatePodModels(p)
					}
					return true
				})
				s.initialSynced.Store(true)
				time.Sleep(uppdateInterval)
			}
		}
	}()
}

func (s *store) AddOrUpdatePod(pod *corev1.Pod, ) error {

	podName := types.NamespacedName{
		Namespace: pod.Namespace,
		Name:      pod.Name,
	}

	if value, ok := s.pods.Load(podName); ok {

		return nil
	}

	// New pod — create PodInfo and fetch initial metrics.
	newPodInfo := &PodInfo{
		Pod:    pod,
		engine: e,
		//modelServer: newModelServers,
		models: sets.New[string](),
	}
	s.pods.Store(podName, newPodInfo)
	s.updatePodMetrics(newPodInfo)
	s.updatePodModels(newPodInfo)

	return nil
}

func (s *store) updatePodMetrics(pod *PodInfo) {
	if pod.engine == "" {
		klog.Info("failed to find backend engine in pod")
		return
	}

	previousHistogram := getPreviousHistogram(pod)
	gaugeMetrics, histogramMetrics := provider.GetPodMetrics(pod.engine, pod.Pod, previousHistogram)
	if gaugeMetrics != nil {
		pod.GPUCacheUsage = gaugeMetrics[GPUCacheUsage]
		pod.RequestRunningNum = gaugeMetrics[RequestRunningNum]
		pod.RequestWaitingNum = gaugeMetrics[RequestWaitingNum]
		pod.TPOT = gaugeMetrics[TPOT]
		pod.TTFT = gaugeMetrics[TTFT]
	}
	if histogramMetrics != nil {
		pod.TimeToFirstToken = histogramMetrics[TTFT]
		pod.TimePerOutputToken = histogramMetrics[TPOT]
	}
}
