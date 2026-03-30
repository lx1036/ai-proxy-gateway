package inferencepool

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/epp/datastore"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	"github.com/lx1036/gateway/pkg/epp/datalayer"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	v1 "sigs.k8s.io/gateway-api-inference-extension/api/v1"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/common"
)

type InferencePoolReconciler struct {
	client.Client
	Datastore datastore.Datastore
	PoolGKNN  common.GKNN
}

func (c *InferencePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.InferencePool{}).
		Complete(c)
}

func (c *InferencePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pool := &v1.InferencePool{}
	if err := c.Get(ctx, req.NamespacedName, pool); err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("InferencePool not found. Clearing the datastore")
			c.Datastore.Clear()
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get InferencePool - %w", err)
	}

	if pool.DeletionTimestamp != nil {
		klog.Infof("InferencePool is marked for deletion. Clearing the datastore")
		c.Datastore.Clear()
		return ctrl.Result{}, nil
	}

	endpointPool := InferencePoolToEndpointPool(pool)
	if err := c.Datastore.ResyncPods(ctx, c.Client, endpointPool); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update datastore - %w", err)
	}

	return ctrl.Result{}, nil
}

func InferencePoolToEndpointPool(inferencePool *v1.InferencePool) *datalayer.EndpointPool {
	if inferencePool == nil {
		return nil
	}

	selector := make(map[string]string, len(inferencePool.Spec.Selector.MatchLabels))
	for k, v := range inferencePool.Spec.Selector.MatchLabels {
		selector[string(k)] = string(v)
	}

	targetPorts := make([]int, 0, len(inferencePool.Spec.TargetPorts))
	for _, p := range inferencePool.Spec.TargetPorts {
		targetPorts = append(targetPorts, int(p.Number))
	}

	return &datalayer.EndpointPool{
		Name:        inferencePool.Name,
		Namespace:   inferencePool.Namespace,
		Selector:    selector,
		TargetPorts: targetPorts,
	}
}
