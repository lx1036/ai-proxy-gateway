package inferencepool

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/epp/utils"
	"github.com/lx1036/gateway/pkg/llmrouter/datastore"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	// PoolLabelKey pod label: {networking.lx1036.ai: inferencepool1}
	PoolLabelKey = "networking.lx1036.ai"
)

type PodReconciler struct {
	client.Client
	Datastore *datastore.Store
}

func (c *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	/**
	InferencePool 里指定了 selector，所以这里会过滤出来该 selector 的 Pods
	  selector:
	    matchLabels:
	      app: envoy-ai-gateway-basic-testupstream
	*/
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			pod, ok := object.(*corev1.Pod)
			if !ok {
				return false
			}

			if pod.Labels == nil {
				return false
			}

			poolName, ok := pod.Labels[PoolLabelKey]
			if !ok {
				return false
			}

			if !c.Datastore.HasPool(poolName) {
				return false
			}

			return c.Datastore.PoolLabelsMatch(poolName, pod.GetLabels())
		})).
		Complete(c)
}

// Reconcile 只 watch InferencePool 指定的 pods
func (c *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pod := &corev1.Pod{}
	if err := c.Get(ctx, req.NamespacedName, pod); err != nil {
		// delete pod，则从 datastore 中删除该 pod
		if apierrors.IsNotFound(err) {
			klog.Infof("pod %s is deleted from the store", req.NamespacedName.String())
			c.Datastore.PodDelete(req.Name)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get pod - %w", err)
	}

	if !utils.IsPodReady(pod) || !c.Datastore.PoolLabelsMatch(pod.Labels) { // 重新 c.Datastore.PoolLabelsMatch 匹配 new pod label 被删除
		klog.Infof("pod is removed")
		c.Datastore.PodDelete(pod.Name)
	} else {
		c.Datastore.PodUpdateOrAddIfNotExist(pod)
	}

	return ctrl.Result{}, nil
}
