package stormservice

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	orchestrationv1alpha1 "github.com/vllm-project/aibrix/api/orchestration/v1alpha1"
)

// stormservice -> roleset -> podset -> pods

type StormServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Record record.EventRecorder
}

func NewStormServiceReconciler(mgr manager.Manager) *StormServiceReconciler {
	return &StormServiceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Record: mgr.GetEventRecorderFor("stormservice-controller"),
	}
}

func (r *StormServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// PodGroupSize: StormService -> RoleSet -> PodSet -> Pods()
	// Stateful/Stateless: StormService -> RoleSet -> Pods
	return ctrl.NewControllerManagedBy(mgr).
		For(&orchestrationv1alpha1.StormService{}).
		Owns(&orchestrationv1alpha1.RoleSet{}).
		Owns(&orchestrationv1alpha1.PodSet{}).
		// 这里为啥没有 Owns(&corev1.Pod{})
		Complete(r)
}

// Reconcile 标配：
// 1. finalizer
func (r *StormServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	stormService := &orchestrationv1alpha1.StormService{}
	if err := r.Get(ctx, req.NamespacedName, stormService); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if stormService.DeletionTimestamp != nil {
		// TODO: finalize
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(stormService, "stormservice.finalizers.vllm.io") {
		// TODO: add finalizer
	}

}
