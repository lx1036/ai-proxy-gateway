package rolebasedgroup

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	coreapplyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	workloadsv1alpha2 "sigs.k8s.io/rbgs/api/workloads/v1alpha2"
)

type PodReconciler struct {
	scheme *runtime.Scheme
	client client.Client
}

func NewPodReconciler(scheme *runtime.Scheme, client client.Client) *PodReconciler {
	return &PodReconciler{
		scheme: scheme,
		client: client,
	}
}

func (r *PodReconciler) ConstructPodTemplateSpecApplyConfiguration(
	ctx context.Context,
	role *workloadsv1alpha2.RoleSpec,
	matchLabels map[string]string,
) (*coreapplyv1.PodTemplateSpecApplyConfiguration, error) {
	var podTemplateSpec corev1.PodTemplateSpec
	if role.GetTemplate() != nil {
		podTemplateSpec = *role.GetTemplate().DeepCopy()
	}

	podAnnotations := podTemplateSpec.Annotations
	if podAnnotations == nil {
		podAnnotations = make(map[string]string)
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&podTemplateSpec)
	if err != nil {
		return nil, err
	}

	// corev1.PodTemplateSpec -> coreapplyv1.PodTemplateSpecApplyConfiguration
	var podTemplateSpecApplyConfiguration *coreapplyv1.PodTemplateSpecApplyConfiguration
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj, &podTemplateSpecApplyConfiguration)
	if err != nil {
		return nil, err
	}

	podTemplateSpecApplyConfiguration.WithLabels(matchLabels)
	podTemplateSpecApplyConfiguration.WithAnnotations(podAnnotations)

	return podTemplateSpecApplyConfiguration, nil
}
