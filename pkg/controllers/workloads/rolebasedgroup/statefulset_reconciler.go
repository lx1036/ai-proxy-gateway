package rolebasedgroup

import (
	"context"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	metaapplyv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1alpha2 "sigs.k8s.io/rbgs/api/workloads/v1alpha2"
)

type StatefulSetReconciler struct {
	scheme *runtime.Scheme
	client client.Client
}

func (r *StatefulSetReconciler) Reconciler() error {
	if err := r.reconcileStatefulSet(ctx, rbg, role, rollingUpdateStrategy, revisionKey); err != nil {
		return err
	}

	return NewServiceReconciler(r.client).reconcileHeadlessService(ctx, rbg, role)
}

func (r *StatefulSetReconciler) reconcileStatefulSet(
	ctx context.Context,
	rbg *workloadsv1alpha2.RoleBasedGroup,
	role *workloadsv1alpha2.RoleSpec,
) error {
	oldSts := &appsv1.StatefulSet{}
	err = r.client.Get(ctx, types.NamespacedName{Name: rbg.GetWorkloadName(role), Namespace: rbg.Namespace}, oldSts)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// create or update
	statefulSetApplyConfiguration, err := r.constructStatefulSetApplyConfiguration(ctx, rbg, role, oldSts, revisionKey)
	if err != nil {
		logger.Error(err, "Failed to construct statefulset apply configuration")
		return err
	}

	// appsapplyv1.StatefulSetApplyConfiguration -> appsv1.StatefulSet
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&statefulSetApplyConfiguration)

	newSts := &appsv1.StatefulSet{}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj, newSts); err != nil {
		return fmt.Errorf("convert stsApplyConfig to sts error: %s", err.Error())
	}

}

func (r *StatefulSetReconciler) constructStatefulSetApplyConfiguration(
	ctx context.Context,
	rbg *workloadsv1alpha2.RoleBasedGroup,
	role *workloadsv1alpha2.RoleSpec,
) (*appsapplyv1.StatefulSetApplyConfiguration, error) {
	matchLabels := rbg.GetCommonLabelsFromRole(role)

	appsv1.StatefulSet{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       appsv1.StatefulSetSpec{},
		Status:     appsv1.StatefulSetStatus{},
	}

	// construct statefulset apply configuration
	podReconciler := NewPodReconciler(r.scheme, r.client)
	podTemplateApplyConfiguration, err := podReconciler.ConstructPodTemplateSpecApplyConfiguration(ctx, role, matchLabels)

	statefulsetSvcName := rbg.GetServiceName(role)
	statefulSetApplyConfiguration := appsapplyv1.StatefulSet(rbg.GetWorkloadName(role), rbg.Namespace). // rbgName-roleName1
		WithLabels(labels.Merge(matchLabels, maps.Clone(role.Labels))).
		WithAnnotations(labels.Merge(rbg.GetCommonAnnotationsFromRole(role), maps.Clone(role.Annotations))).
		WithOwnerReferences(
			metaapplyv1.OwnerReference().
				WithAPIVersion(rbg.APIVersion).
				WithKind(rbg.Kind).
				WithName(rbg.Name).
				WithUID(rbg.UID).
				WithController(true).
				WithBlockOwnerDeletion(true),
		).
		WithSpec(
			appsapplyv1.StatefulSetSpec().
				WithServiceName(statefulsetSvcName).
				WithReplicas(*role.Replicas).
				WithPodManagementPolicy(appsv1.PodManagementPolicyType(role.PodManagementPolicy)).
				WithSelector(metaapplyv1.LabelSelector().WithMatchLabels(matchLabels)).
				WithMinReadySeconds(role.MinReadySeconds).
				WithTemplate(podTemplateApplyConfiguration),
		)

	return statefulSetApplyConfiguration, nil
}
