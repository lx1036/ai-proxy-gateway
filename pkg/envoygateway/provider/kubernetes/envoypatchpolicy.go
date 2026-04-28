package kubernetes

import (
	"context"

	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/lx1036/gateway/pkg/envoygateway/message"
)

func (gatewayAPIReconciler *GatewayAPIReconciler) processEnvoyPatchPolicy(ctx context.Context, resources *message.GatewayAPIResource) {
	// INFO: no indexer???
	var envoyPatchPolicyList envoygatewayv1alpha1.EnvoyPatchPolicyList
	if err := gatewayAPIReconciler.mgr.GetClient().List(ctx, &envoyPatchPolicyList); err != nil {

	}

	for i, _ := range envoyPatchPolicyList.Items {
		envoyPatchPolicy := &envoyPatchPolicyList.Items[i]

		resources.EnvoyPatchPolicies = append(resources.EnvoyPatchPolicies, envoyPatchPolicy)

	}

}
