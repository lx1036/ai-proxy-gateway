package datastore

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/epp/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"net"
	"strconv"
	"sync"

	"github.com/lx1036/gateway/pkg/epp/datalayer"
	"github.com/lx1036/gateway/pkg/epp/datalayer/backend/metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/util/sets"
)

type Datastore struct {
	mu sync.RWMutex

	//pools map[string]*v1.InferencePool // 直接保存 v1.InferencePool 对象，占内存太大
	pools map[string]*datalayer.EndpointPool // 直接保存 v1.InferencePool 对象，占内存太大

	parentCtx         context.Context
	podMetricsFactory *metrics.PodMetricsFactory

	pool *datalayer.EndpointPool

	// key: types.NamespacedName, value: *backend/metrics.PodMetrics
	pods *sync.Map
}

type DatastoreOption func(*Datastore)

func NewDatastore(parentCtx context.Context, podMetricsFactory *metrics.PodMetricsFactory) *Datastore {
	store := &Datastore{
		mu: sync.RWMutex{},

		parentCtx:         parentCtx,
		podMetricsFactory: podMetricsFactory,

		pods: &sync.Map{},

		pool: nil,
	}

	return store
}

func (ds *Datastore) PodUpdateOrAddIfNotExist(pod *corev1.Pod) bool {
	labels := make(map[string]string, len(pod.GetLabels()))
	for key, value := range pod.GetLabels() {
		labels[key] = value
	}
	// TargetPorts defines a list of ports that are exposed by this InferencePool.
	// Every port will be treated as a distinctive endpoint by EPP,
	var metadatas []*datalayer.EndpointMetadata
	for idx, port := range ds.pool.TargetPorts {
		metadatas = append(metadatas,
			&datalayer.EndpointMetadata{
				NamespacedName: types.NamespacedName{
					Name:      pod.Name + "-rank-" + strconv.Itoa(idx),
					Namespace: pod.Namespace,
				},
				PodName:     pod.Name,
				Address:     pod.Status.PodIP,
				Port:        strconv.Itoa(port),
				MetricsHost: net.JoinHostPort(pod.Status.PodIP, strconv.Itoa(port)),
				Labels:      labels,
			})
	}

	existed := true
	for _, metadata := range metadatas {
		existing, ok := ds.pods.Load(metadata.NamespacedName)
		if !ok {
			// start gorouting loop to fetch pod metrics
			podMetrics := ds.podMetricsFactory.NewEndpoint(ds.parentCtx, metadata)
			ds.pods.Store(metadata.NamespacedName, podMetrics)
			existed = false
		} else {
			podMetrics := existing.(*metrics.PodMetrics)
			podMetrics.UpdateMetadata(metadata)
		}
	}

	return existed
}

// PodDelete 1. delete from sync.Map 2. release gorouting loop(fetch pod metrics)
func (ds *Datastore) PodDelete(podName string) {
	ds.pods.Range(func(key, value any) bool {
		podMetrics := value.(*metrics.PodMetrics)
		if podMetrics.GetMetadata().PodName == podName {
			// delete from sync.Map
			ds.pods.Delete(key)
			// release gorouting loop(fetch pod metrics)
			ds.podMetricsFactory.ReleaseEndpoint(podMetrics)
		}

		return true
	})
}

func (ds *Datastore) ResyncPods(ctx context.Context, c client.Client, endpointPool *datalayer.EndpointPool) error {
	if endpointPool == nil {
		ds.Clear()
		return nil
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	oldEndpointPool := ds.pool
	ds.pool = endpointPool

	// 首次更新或者后期有更新
	if oldEndpointPool == nil || !labels.Equals(oldEndpointPool.Selector, endpointPool.Selector) {
		klog.Infof("inference pool selector is updated: %v", endpointPool.Selector)

		podList := &corev1.PodList{}
		if err := c.List(ctx, podList, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(ds.pool.Selector),
			Namespace:     ds.pool.Namespace,
		}); err != nil {
			return fmt.Errorf("failed to list pods: %v", err)
		}

		activePods := sets.New[string]()
		for _, pod := range podList.Items {
			// INFO: 所以 InferencePool 得指定好已经 Ready 的 pods，不管 pods 是由 Deployment/LWS/StormService 创建的
			if !utils.IsPodReady(&pod) {
				continue
			}

			namespacedName := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
			activePods.Insert(pod.Name)
			klog.Infof("start metrics loop cycle for pod %s", namespacedName.String())
			if ds.PodUpdateOrAddIfNotExist(&pod) {
				klog.Infof("pod %s is already existed", namespacedName.String())
			} else {
				klog.Infof("pod %s is new added", namespacedName.String())
			}
		}

		ds.pods.Range(func(key, value any) bool {
			podMetrics := value.(*metrics.PodMetrics)
			if !activePods.Has(podMetrics.GetMetadata().PodName) {
				klog.Infof("pod %s is deleted from datastore, and released from endpoint metrics loop cycle", podMetrics.GetMetadata().NamespacedName.String())
				ds.PodDelete(podMetrics.GetMetadata().PodName)
			}

			return true
		})
	}

	return nil
}

func (ds *Datastore) Clear() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.pool = nil

	// stop all pods go routines before clearing the pods map.
	ds.pods.Range(func(key, value any) bool {
		podMetrics := value.(*metrics.PodMetrics)
		ds.PodDelete(podMetrics.GetMetadata().PodName)

		return true
	})
	ds.pods.Clear()
}
