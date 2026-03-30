package lws

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	leaderworkersetv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
)

// LeaderWorkerSetReconciler reconciles a LeaderWorkerSet object
type LeaderWorkerSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Record record.EventRecorder
}

func NewLeaderWorkerSetReconciler(mgr manager.Manager) *LeaderWorkerSetReconciler {
	return &LeaderWorkerSetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Record: mgr.GetEventRecorderFor("lws-controller"),
	}
}

func (r *LeaderWorkerSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&leaderworkersetv1.LeaderWorkerSet{}). // lws -> StatefulSet(service) -> pods
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}). // 同时 owns service
		Watches(&appsv1.StatefulSet{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request { // TODO: 这个是在做什么？
			return []reconcile.Request{
				{
					// sts: telellm-deepseek-r1 和 telellm-deepseek-r1-0
					NamespacedName: types.NamespacedName{
						Name:      object.GetLabels()[leaderworkersetv1.SetNameLabelKey], // 这里的 object 应该是 StatefulSet 资源对象
						Namespace: object.GetNamespace(),
					},
				},
			}
		})).
		Complete(r)
}

func (r *LeaderWorkerSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lws := &leaderworkersetv1.LeaderWorkerSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, lws); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !lws.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	leaderSts, err := r.getLeaderStatefulSet(ctx, lws) // leader sts 和 lws name 一样
	if err != nil {
		klog.Errorf("fail to fetch leader statefulset: %v", err)
		return ctrl.Result{}, err
	}

	if leaderSts != nil && leaderSts.DeletionTimestamp != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	/**
	updateStrategy:
	    type: RollingUpdate
	    rollingUpdate:
	      partition: 0
	*/

	/**
	1. partition update strategy
	@see https://github.com/kubernetes-sigs/lws/blob/main/keps/511-partition-update/README.md
	作用：
	(1) lws 具备执行分区更新的能力，从而促进金丝雀发布或灰色发布
	(2) PD 分离场景下，可以对齐 P 和 D 的版本比率
	*/

	leaderSts.Spec.UpdateStrategy.RollingUpdate.Partition

}

func (r *LeaderWorkerSetReconciler) getLeaderStatefulSet(ctx context.Context, lws *leaderworkersetv1.LeaderWorkerSet) (*appsv1.StatefulSet, error) {
	sts := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: lws.Name, Namespace: lws.Namespace}, sts)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return sts, nil
}

func (r *LeaderWorkerSetReconciler) partitionUpdate(ctx context.Context, lws *leaderworkersetv1.LeaderWorkerSet) {

}
