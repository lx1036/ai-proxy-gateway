package kubernetes

import (
	"context"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)


func (gatewayAPIReconciler *GatewayAPIReconciler) processBackendRef(ctx context.Context, backendRef gatewayapiv1.BackendObjectReference) {

	// TODO: ReferenceGrant
}
