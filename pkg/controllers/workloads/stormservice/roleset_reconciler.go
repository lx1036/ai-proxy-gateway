package stormservice

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	orchestrationv1alpha1 "github.com/vllm-project/aibrix/api/orchestration/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type RoleSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Record record.EventRecorder
}

func NewRoleSetReconciler(mgr manager.Manager) *RoleSetReconciler {
	return &RoleSetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Record: mgr.GetEventRecorderFor("roleset-controller"),
	}
}

func (r *RoleSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// PodGroupSize: StormService -> RoleSet -> PodSet -> Pods()
	// Stateful/Stateless: StormService -> RoleSet -> Pods
	return ctrl.NewControllerManagedBy(mgr).
		For(&orchestrationv1alpha1.RoleSet{}).
		//Owns(&orchestrationv1alpha1.PodSet{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func (r *RoleSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	roleSet := &orchestrationv1alpha1.RoleSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, roleSet); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if roleSet.DeletionTimestamp != nil {
		// TODO: finalize
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(roleSet, "roleSet.finalizers.vllm.io") {
		// TODO: add finalizer
	}

}
