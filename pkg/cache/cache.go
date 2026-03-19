package cache

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/cache/collector"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sync"
	"time"

	k8sCache "k8s.io/client-go/tools/cache"
)

type Cache interface {
	ListPodsByModel(model string) (v1.PodList, error)
	//ListPodsByModel(model any) (interface{}, interface{})
}

type cache struct {
	mutex sync.RWMutex // Read-write mutex for concurrency safety

	podMetricsWorkerCount int
	podMetricsJobs        chan *Pod

	models sync.Map[string, string]

	collector *collector.Collector
}

func NewCache() Cache {
	c := &cache{
		podMetricsWorkerCount: 10,
		podMetricsJobs:        make(chan *Pod, 100),
	}

	for w := 0; w < c.podMetricsWorkerCount; w++ {
		go c.worker(c.podMetricsJobs)
	}

	return c
}

type Pod struct {
	*v1.Pod
}

func (c *cache) worker(jobs <-chan *Pod) {

	for pod := range jobs {

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		podMetricPort := getPodMetricPort(pod)
		endpoint := fmt.Sprintf("%s:%d", pod.Status.PodIP, podMetricPort)

		result, err := c.collector.Collect(ctx, endpoint)

		c.updatePodMetrics()

		cancel()
	}
}

func (c *cache) ListPodsByModel(model string) (v1.PodList, error) {

	meta, ok := c.models.Load(modelName)

}

func (c *cache) initInformers() error {

	k8sClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	factory := informers.NewSharedInformerFactory(k8sClientSet, 0)
	podInformer := factory.Core().V1().Pods().Informer()
	if _, err := podInformer.AddEventHandler(k8sCache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		UpdateFunc: c.updatePod,
		DeleteFunc: c.deletePod,
	}); err != nil {
		return err
	}

	factory.Start(stopCh)

	if !k8sCache.WaitForCacheSync(stopCh, podInformer.HasSynced) {
		return fmt.Errorf("timed out waiting for caches to sync")
	}

	klog.Infof("Cache initialization completed. Models: %v", c.ListModels())

	return nil
}
