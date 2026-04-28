package kubernetes

import (
	"context"
	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/lx1036/gateway/pkg/envoygateway/message"
)

func (gatewayAPIReconciler *GatewayAPIReconciler) processEnvoyExtensionPolicy(ctx context.Context, resources *message.GatewayAPIResource) {


	// INFO: no indexer???
	var envoyExtensionPolicyList envoygatewayv1alpha1.EnvoyExtensionPolicyList
	if err := gatewayAPIReconciler.mgr.GetClient().List(ctx, &envoyExtensionPolicyList); err != nil {

	}



	for i, _ := range envoyExtensionPolicyList.Items {
		envoyExtensionPolicy := &envoyExtensionPolicyList.Items[i]

		resources.EnvoyExtensionPolicies = append(resources.EnvoyExtensionPolicies, envoyExtensionPolicy)

	}




	gatewayAPIReconciler.processEnvoyExtensionPolicyObjectRefs(ctx, resources)
}


func (gatewayAPIReconciler *GatewayAPIReconciler) processEnvoyExtensionPolicyObjectRefs(ctx context.Context, resources *message.GatewayAPIResource) {

	for _, policy := range resources.EnvoyExtensionPolicies {

		for _, extProc := range policy.Spec.ExtProc {

			for _, backendRef := range extProc.BackendRefs {

				gatewayAPIReconciler.processBackendRef(ctx, backendRef.BackendObjectReference)
			}



		}

	}

}
