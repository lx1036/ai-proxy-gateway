package lws

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	coreapplyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	metaapplyv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			if lws, ok := object.(*leaderworkersetv1.LeaderWorkerSet); ok {
				_, exist := lws.Labels["lx1036"]
				if exist {
					klog.Infof("[lws_controller] watch LeaderWorkerSet resource: %s/%s", lws.Namespace, lws.Name)
				}
				return exist
			}
			if statefulSet, ok := object.(*appsv1.StatefulSet); ok {
				_, exist := statefulSet.Labels[leaderworkersetv1.SetNameLabelKey]
				if exist {
					klog.Infof("[lws_controller] watch appsv1.StatefulSet resource: %s/%s", statefulSet.Namespace, statefulSet.Name)
				}
				return exist
			}

			return false
		})).
		//Watches(&appsv1.StatefulSet{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request { // TODO: 这个是在做什么？
		//	return []reconcile.Request{
		//		{
		//			// sts: telellm-deepseek-r1 和 telellm-deepseek-r1-0
		//			NamespacedName: types.NamespacedName{
		//				Name:      object.GetLabels()[leaderworkersetv1.SetNameLabelKey], // 这里的 object 应该是 StatefulSet 资源对象
		//				Namespace: object.GetNamespace(),
		//			},
		//		},
		//	}
		//})).
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

    /////////////////////// for debug ///////////////////////
	// INFO: fail to list statefulset: Index with name field:.metadata.controller does not exist
	var stsList appsv1.StatefulSetList
	err := r.List(ctx, &stsList, client.InNamespace(lws.Namespace), client.MatchingFields{".metadata.controller": string(lws.UID)})
	if err != nil {
		klog.Errorf("fail to list statefulset: %v", err)
		return ctrl.Result{}, err
	}
	if len(stsList.Items) == 0 {
		klog.Infof("No StatefulSet found for LeaderWorkerSet %s/%s", lws.Namespace, lws.Name)
		return ctrl.Result{}, nil
	}
	for _, item := range stsList.Items {
		klog.Infof("StatefulSet %s/%s found for LeaderWorkerSet %s/%s", item.Namespace, item.Name, lws.Namespace, lws.Name)
	}
	//if len(stsList.Items) >= 1 {
	//	klog.Errorf("More than one StatefulSet found for LeaderWorkerSet %s/%s", lws.Namespace, lws.Name)
	//	return ctrl.Result{}, nil
	//}
	/////////////////////// for debug ///////////////////////

	leaderSts := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: lws.Name, Namespace: lws.Namespace}, leaderSts)
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("fail to fetch statefulset: %v", err)
		return ctrl.Result{}, err
	}

	if leaderSts != nil && leaderSts.DeletionTimestamp != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	revisionKey := ""
	if leaderSts != nil {
		revisionKey = GetRevisionKey(leaderSts)
	}

	revision, err := GetOrCreateRevision(ctx, r.Client, lws, revisionKey)
	if err != nil {
		klog.Errorf("fail to get or create revision: %v", err)
		return ctrl.Result{}, err
	}

	var updatedRevision *appsv1.ControllerRevision
	if leaderSts != nil {
		currentRevision, err := NewRevision(ctx, r.Client, lws, "")
		if err != nil {
			klog.Errorf("fail to get or create revision: %v", err)
			return ctrl.Result{}, err
		}
		if currentRevision != revision {
			updatedRevision = currentRevision
		}
	}

	lwsUpdated := updatedRevision != nil
	if lwsUpdated {
		err = r.Client.Create(ctx, updatedRevision)
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

	// 2. create leader sts
	stsApplyConfiguration, err := r.constructLeaderStatefulSetApplyConfiguration(lws, revision)

	// INFO: leader sts 的 owner 是 lws
	if err = setControllerReferenceWithStatefulSet(lws, stsApplyConfiguration, r.Scheme); err != nil {
		klog.Errorf("set controller reference err: %v", err)
		return ctrl.Result{}, err
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(stsApplyConfiguration)
	if err != nil {
		return ctrl.Result{}, err
	}
	workerStatefulSet := &unstructured.Unstructured{
		Object: obj,
	}
	// 使用 kubectl apply -f
	err = r.Patch(ctx, workerStatefulSet, client.Apply, &client.PatchOptions{
		FieldManager: "lws",
		Force:        ptr.To[bool](true),
	})

	klog.Infof("Leader Reconcile completed.")
	return ctrl.Result{}, nil
}

func (r *LeaderWorkerSetReconciler) constructLeaderStatefulSetApplyConfiguration(lws *leaderworkersetv1.LeaderWorkerSet,
	revision *appsv1.ControllerRevision) (*appsapplyv1.StatefulSetApplyConfiguration, error) {

	// INFO: 这里容许 LeaderTemplate 为空，这样来表示 单节点模式，替换掉 Deployment 部署单机 LLM 推理服务
	var podTemplateSpec corev1.PodTemplateSpec
	if lws.Spec.LeaderWorkerTemplate.LeaderTemplate != nil {
		podTemplateSpec = *lws.Spec.LeaderWorkerTemplate.LeaderTemplate.DeepCopy()
	} else {
		podTemplateSpec = *lws.Spec.LeaderWorkerTemplate.WorkerTemplate.DeepCopy()
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&podTemplateSpec)
	if err != nil {
		return nil, err
	}
	var podTemplateApplyConfiguration coreapplyv1.PodTemplateSpecApplyConfiguration
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj, &podTemplateApplyConfiguration)
	if err != nil {
		return nil, err
	}

	stsLabels := map[string]string{
		leaderworkersetv1.SetNameLabelKey: lws.Name,
		leaderworkersetv1.RevisionKey:     GetRevisionKey(revision),
	}
	stsAnnotations := map[string]string{
		leaderworkersetv1.ReplicasAnnotationKey: strconv.Itoa(int(*lws.Spec.Replicas)),
	}
	matchLabels := map[string]string{
		leaderworkersetv1.SetNameLabelKey:     lws.Name,
		leaderworkersetv1.WorkerIndexLabelKey: "0",
	}

	podTemplateApplyConfiguration.WithLabels(matchLabels).WithLabels(map[string]string{
		leaderworkersetv1.RevisionKey: GetRevisionKey(revision),
	})

	podAnnotations := map[string]string{
		leaderworkersetv1.SizeAnnotationKey: strconv.Itoa(int(*lws.Spec.LeaderWorkerTemplate.Size)),
	}
	podTemplateApplyConfiguration.WithAnnotations(podAnnotations)

	statefulSetConfig := appsapplyv1.StatefulSet(lws.Name, lws.Namespace).
		WithLabels(stsLabels).
		WithAnnotations(stsAnnotations).
		WithSpec(
			appsapplyv1.StatefulSetSpec().
				WithServiceName(lws.Name).
				WithReplicas(*lws.Spec.Replicas). // INFO: 直接使用 lws 的 partition. 有 replicas 个 leader pods
				WithSelector(metaapplyv1.LabelSelector().WithMatchLabels(matchLabels)).
				WithTemplate(&podTemplateApplyConfiguration).
				WithPodManagementPolicy(appsv1.ParallelPodManagement). // OrderedReady: 依次创建；Parallel: 并行创建. worker pod 可以 Parallel
				WithUpdateStrategy(appsapplyv1.StatefulSetUpdateStrategy().
					WithType(appsv1.StatefulSetUpdateStrategyType(lws.Spec.RolloutStrategy.Type)).
					WithRollingUpdate(appsapplyv1.RollingUpdateStatefulSetStrategy().
						WithPartition(*lws.Spec.RolloutStrategy.RollingUpdateConfiguration.Partition). // INFO: 直接使用 lws 的 partition
						WithMaxUnavailable(lws.Spec.RolloutStrategy.RollingUpdateConfiguration.MaxUnavailable),
					),
				),
		)

	// skip pvc

	return statefulSetConfig, nil
}

