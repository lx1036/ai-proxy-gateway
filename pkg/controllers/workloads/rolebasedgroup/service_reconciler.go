package rolebasedgroup

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1alpha2 "sigs.k8s.io/rbgs/api/workloads/v1alpha2"
)

// INFO: 核心逻辑：就是创建 headless svc

type ServiceReconciler struct {
	Client client.Client
	Record record.EventRecorder
}

func NewServiceReconciler(mgr manager.Manager) *ServiceReconciler {
	return &ServiceReconciler{
		Client: mgr.GetClient(),
		Record: mgr.GetEventRecorderFor("rbg-controller"),
	}
}

func (r *ServiceReconciler) reconcileHeadlessService(
	ctx context.Context,
	rbg *workloadsv1alpha2.RoleBasedGroup,
	role *workloadsv1alpha2.RoleSpec,
) {

	obj := &appsv1.StatefulSet{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: workloadName, Namespace: rbg.Namespace}, obj)





	svc := corev1.Service{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: rbg.GetServiceName(role),
			Namespace: rbg.Namespace,
			Labels: ,
			Annotations: ,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, appsv1.SchemeGroupVersion.WithKind(obj.Kind)),
			},
		},
		Spec:       corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			Selector: map[string]string{
				"rbg.workloads.x-k8s.io/group-name": rbg.Name,
				"rbg.workloads.x-k8s.io/role-name": role.Name,
			},
			PublishNotReadyAddresses: true,

		},
		Status:     corev1.ServiceStatus{},
	}

	oldSvc := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: svcName, Namespace: rbg.Namespace}, oldSvc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// create svc
			if err = r.Client.Create(ctx, &svc); err != nil {
				return err
			}

			r.Record.Eventf(rbg, corev1.EventTypeNormal, "HeadlessServiceCreated", "Service %s/%s created", svc.Namespace, svc.Name)

			return nil
		}

		return err
	}

	// not equal
	if !isServiceEqual() {
		if err = r.Client.Update(ctx, &svc); err != nil {
			return err
		}

		r.Record.Eventf(rbg, corev1.EventTypeNormal, "HeadlessServiceUpdated", "Service %s/%s updated", svc.Namespace, svc.Name)
	}

	return nil
}
