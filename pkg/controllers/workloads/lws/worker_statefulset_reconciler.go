package lws

import (
	"context"
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	coreapplyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	metaapplyv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	leaderworkersetv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
)

/**
功能点：
1. 根据 leader pod 来创建 worker StatefulSet。在 rolling update 的时候，leader pod 会根据 revision 删除，owner 的 worker StatefulSet 也会被级联删除。
leader pod 会触发 WorkerStatefulSetReconciler.Reconcile() 走 Get-or-Create 逻辑
*/

type WorkerStatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Record record.EventRecorder
}

func NewWorkerStatefulSetReconciler(mgr manager.Manager) *WorkerStatefulSetReconciler {
	return &WorkerStatefulSetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Record: mgr.GetEventRecorderFor("statefulset-controller"),
	}
}

func (r *WorkerStatefulSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	/**
	1. Pod/StatefulSet 都会 ListAndWatch 过来，WithEventFilter(obj) obj 会是 Pod/StatefulSet。因为也 Owns(&StatefulSet) 了。
	*/

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}). // all namespace pods
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			if pod, ok := object.(*corev1.Pod); ok {
				_, exist := pod.Labels[leaderworkersetv1.SetNameLabelKey]
				if exist {
					klog.Infof("[sts_controller] watch corev1.Pod resource: %s/%s", pod.Namespace, pod.Name)
				}
				return exist
			}
			if statefulSet, ok := object.(*appsv1.StatefulSet); ok {
				_, exist := statefulSet.Labels[leaderworkersetv1.SetNameLabelKey]
				if exist {
					klog.Infof("[sts_controller] watch appsv1.StatefulSet resource: %s/%s", statefulSet.Namespace, statefulSet.Name)
				}
				return exist
			}

			// for debug
			if svc, ok := object.(*corev1.Service); ok {
				klog.Infof("[sts_controller] watch corev1.Service resource: %s/%s", svc.Namespace, svc.Name)
				return true
			}

			return false
		})).
		Owns(&appsv1.StatefulSet{}).
		//Owns(&corev1.Service{}).
		Complete(r)
}

func (r *WorkerStatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.Infof("req: %s/%s", req.Namespace, req.Name)

	var leaderPod corev1.Pod
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, &leaderPod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// get the leaderWorkerSet name
	lwsName := leaderPod.Labels[leaderworkersetv1.SetNameLabelKey]
	if len(lwsName) == 0 {
		return ctrl.Result{}, errors.New("leaderworkersetv1.sigs.k8s.io/name label is unexpected missing")
	}
	if _, exist := leaderPod.Labels[leaderworkersetv1.WorkerIndexLabelKey]; !exist {
		return ctrl.Result{}, errors.New("leaderworkersetv1.sigs.k8s.io/worker-index label is unexpected missing")
	}

	// 必须是 leader leaderPod -> worker sts
	if !LeaderPod(leaderPod) {
		return ctrl.Result{}, nil
	}

	if leaderPod.Annotations[leaderworkersetv1.LeaderPodNameAnnotationKey] != "" {
		return ctrl.Result{}, nil
	}

	if leaderPod.DeletionTimestamp != nil {
		// leader leaderPod is deleting
		klog.Infof("[statefulset_controller]leader leaderPod is deleting for skip creating worker sts: %s/%s", leaderPod.Namespace, leaderPod.Name)
		return ctrl.Result{}, nil
	}

	//get the leaderWorkerSet object
	var leaderWorkerSet leaderworkersetv1.LeaderWorkerSet
	if err := r.Get(ctx, types.NamespacedName{Name: lwsName, Namespace: leaderPod.Namespace}, &leaderWorkerSet); err != nil {
		// If lws not found, it's mostly because deleted, ignore the error as Pods will be GCed finally.
		// no kind is registered for the type v1.LeaderWorkerSet in scheme "k8s.io/client-go/kubernetes/scheme/register.go:84"
		//klog.Errorf("get lws err: %v", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Once size = 1, no need to create worker statefulSets.
	if *leaderWorkerSet.Spec.LeaderWorkerTemplate.Size == 1 {
		return ctrl.Result{}, nil
	}

	revisionKey := GetRevisionKey(&leaderPod)
	currentControllerRevision, err := GetRevision(ctx, r.Client, &leaderWorkerSet, revisionKey)
	if err != nil {
		klog.Errorf("get revision err: %v", err)
		return ctrl.Result{}, err
	}
	if currentControllerRevision == nil {
		klog.Infof("Revision has not been created yet, requeing reconciler for leaderPod %s", leaderPod.Name)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
	}

	stsApplyConfiguration, err := constructWorkerStatefulSetApplyConfiguration(leaderPod, leaderWorkerSet, currentControllerRevision)
	if err != nil {
		klog.Errorf("construct worker statefulSet err: %v", err)
		return ctrl.Result{}, err
	}
	// INFO: worker sts 的 owner 是对应的 leaderPod
	if err = setControllerReferenceWithStatefulSet(&leaderPod, stsApplyConfiguration, r.Scheme); err != nil {
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

	// INFO: 这里的设计是，worker sts 不需要 Apply Patch，直接就是 Get-or-Create. 那如果更新了 workerTemplate，岂不是不会更新 worker sts

	// Use server side apply and add fieldmanagaer to the lws owned fields
	// If there are conflicts in the fields owned by the lws controller, lws will obtain the ownership and force override
	// these fields to the ones desired by the lws controller. These fields are specified in the StatefulSetApplyConfiguration
	/*err = r.Patch(ctx, workerStatefulSet, client.Apply, &client.PatchOptions{
		FieldManager: fieldManager,
		Force:        ptr.To[bool](true),
	})
	if err != nil {
		return ctrl.Result{}, err
	}*/

	var workerSts appsv1.StatefulSet
	if err := r.Get(ctx, types.NamespacedName{Name: leaderPod.Name, Namespace: leaderPod.Namespace}, &workerSts); err != nil {
		if client.IgnoreNotFound(err) != nil {
			klog.Errorf("[statefulset_controller]get worker sts err: %v", err)
			return ctrl.Result{}, err
		}
		// worker sts not found, create it
		klog.Infof("[statefulset_controller]worker sts not found, create it: %s/%s", leaderPod.Namespace, leaderPod.Name)
		if err = r.Client.Create(ctx, workerStatefulSet); err != nil {
			klog.Errorf("[statefulset_controller]create worker sts err: %v", err)
			r.Record.Eventf(&leaderWorkerSet, corev1.EventTypeWarning, "StatefulSetCreateFailed", "Failed to create worker statefulSet: %v", err)
			return ctrl.Result{}, err
		}

		r.Record.Eventf(&leaderWorkerSet, corev1.EventTypeNormal, "StatefulSetCreated", "StatefulSet %s/%s created", workerSts.Namespace, workerSts.Name)
	}

	klog.Infof("Worker Statefulset Reconcile completed.")
	return ctrl.Result{}, nil

	//var workerSts appsv1.StatefulSet
	//if err := r.Get(ctx, types.NamespacedName{Name: leaderPod.Name, Namespace: leaderPod.Namespace}, &workerSts); err != nil {
	//	if client.IgnoreNotFound(err) != nil {
	//		return ctrl.Result{}, err
	//	}
	//	// worker sts not found, create it
	//	klog.Infof("[statefulset_controller]worker sts not found, create it: %s/%s", leaderPod.Namespace, leaderPod.Name)
	//	//workerSts = appsv1.StatefulSet{}
	//}
	//
	//// 直接从 apiserver 里取值
	//var svc corev1.Service
	//if err := r.Get(ctx, types.NamespacedName{Name: "lws-min1", Namespace: leaderPod.Namespace}, &svc); err != nil {
	//	if client.IgnoreNotFound(err) != nil {
	//		return ctrl.Result{}, err
	//	}
	//	// service not found, create it
	//	klog.Infof("[statefulset_controller]service not found, create it: %s/%s", leaderPod.Namespace, leaderPod.Name)
	//	//svc = corev1.Service{}
	//}
}

func LeaderPod(pod corev1.Pod) bool {
	return pod.Labels[leaderworkersetv1.WorkerIndexLabelKey] == "0"
}

func constructWorkerStatefulSetApplyConfiguration(leaderPod corev1.Pod, lws leaderworkersetv1.LeaderWorkerSet,
	currentRevision *appsv1.ControllerRevision) (*appsapplyv1.StatefulSetApplyConfiguration, error) {

	currentLws, err := ApplyRevision(&lws, currentRevision)
	if err != nil {
		return nil, err
	}

	// INFO: 使用 currentRevision 对应的 lws 的 workerTemplate，而不是最新 lws 的???
	podTemplateSpec := *currentLws.Spec.LeaderWorkerTemplate.WorkerTemplate.DeepCopy()
	//podTemplateSpec := *lws.Spec.LeaderWorkerTemplate.WorkerTemplate.DeepCopy()
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
		leaderworkersetv1.GroupIndexLabelKey:      leaderPod.Labels[leaderworkersetv1.GroupIndexLabelKey],
		leaderworkersetv1.SetNameLabelKey:         lws.Name,
		leaderworkersetv1.GroupUniqueHashLabelKey: leaderPod.Labels[leaderworkersetv1.GroupUniqueHashLabelKey],
		leaderworkersetv1.RevisionKey:             GetRevisionKey(&leaderPod),
	}

	serviceName := leaderPod.Name
	if lws.Spec.NetworkConfig == nil || *lws.Spec.NetworkConfig.SubdomainPolicy == leaderworkersetv1.SubdomainShared {
		serviceName = lws.Name
	}

	matchLabels := map[string]string{
		leaderworkersetv1.GroupIndexLabelKey:      leaderPod.Labels[leaderworkersetv1.GroupIndexLabelKey],
		leaderworkersetv1.SetNameLabelKey:         lws.Name,
		leaderworkersetv1.GroupUniqueHashLabelKey: leaderPod.Labels[leaderworkersetv1.GroupUniqueHashLabelKey],
	}
	podTemplateApplyConfiguration.WithLabels(matchLabels)
	podAnnotations := map[string]string{
		leaderworkersetv1.SizeAnnotationKey:          strconv.Itoa(int(*lws.Spec.LeaderWorkerTemplate.Size)),
		leaderworkersetv1.LeaderPodNameAnnotationKey: leaderPod.Name,
	}
	podTemplateApplyConfiguration.WithAnnotations(podAnnotations)

	// INFO: 注意，这里使用 leader pod name 构造 worker StatefulSet
	statefulSetConfig := appsapplyv1.StatefulSet(leaderPod.Name, leaderPod.Namespace).
		WithLabels(stsLabels).
		//WithAnnotations().
		WithSpec(
			appsapplyv1.StatefulSetSpec().
				WithServiceName(serviceName).
				WithReplicas(*lws.Spec.LeaderWorkerTemplate.Size - 1). // 减去 leader 数量
				WithSelector(metaapplyv1.LabelSelector().WithMatchLabels(matchLabels)).
				WithTemplate(&podTemplateApplyConfiguration).
				WithPodManagementPolicy(appsv1.ParallelPodManagement). // OrderedReady: 依次创建；Parallel: 并行创建. worker pod 可以 Parallel
				// worker pod 索引从 1 开始. 这样 leader-worker 架构 pods 为： leader-0, worker-1, worker-2, worker-3, worker-4
				// lws-min1-0(lws-min1), lws-min1-0-1(lws-min1-0)，而不是 lws-min1-0-0(lws-min1-0)
				WithOrdinals(appsapplyv1.StatefulSetOrdinals().WithStart(1)),
		)

	// skip pvc

	return statefulSetConfig, nil
}

// 函数可以重用
func setControllerReferenceWithStatefulSet(owner metav1.Object, sts *appsapplyv1.StatefulSetApplyConfiguration, scheme *runtime.Scheme) error {
	// Validate the owner.
	ro, ok := owner.(runtime.Object)
	if !ok {
		return fmt.Errorf("%T is not a runtime.Object, cannot call SetOwnerReference", owner)
	}
	gvk, err := apiutil.GVKForObject(ro, scheme)
	if err != nil {
		return err
	}

	sts.WithOwnerReferences(metaapplyv1.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(owner.GetName()).
		WithUID(owner.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true))

	return nil
}
