package rolebasedgroup

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	metaapplyv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"maps"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	workloadsv1alpha2 "sigs.k8s.io/rbgs/api/workloads/v1alpha2"
)

type RoleBasedGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Record record.EventRecorder
}

func NewRoleBasedGroupReconciler(mgr manager.Manager) *RoleBasedGroupReconciler {
	return &RoleBasedGroupReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Record: mgr.GetEventRecorderFor("rbg-controller"),
	}
}

func (r *RoleBasedGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// RBG -> StatefulSet -> Pod
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadsv1alpha2.RoleBasedGroup{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return true
		}))).
		Owns(&appsv1.StatefulSet{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return true // INFO: 作为样板代码 template code
		}))).
		Complete(r)
}

func (r *RoleBasedGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rbg := &workloadsv1alpha2.RoleBasedGroup{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, rbg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !rbg.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

}

// RBG -> StatefulSet
