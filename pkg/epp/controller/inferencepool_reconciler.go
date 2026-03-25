package controller

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/epp/datastore"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

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
	obj := &v1.InferencePool{}
	if err := c.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("InferencePool not found. Clearing the datastore")
			c.Datastore.Clear(obj)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get InferencePool - %w", err)
	}

	if !obj.GetDeletionTimestamp().IsZero() {
		klog.Infof("InferencePool is marked for deletion. Clearing the datastore")
		c.Datastore.Clear(obj)
		return ctrl.Result{}, nil
	}

	if err := c.Datastore.StorePools(obj); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update datastore - %w", err)
	}

	return ctrl.Result{}, nil
}
