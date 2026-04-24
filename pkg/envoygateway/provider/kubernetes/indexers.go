package kubernetes

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const  (
	gatewayHTTPRouteIndex            = "gatewayHTTPRouteIndex"

)

func addHTTPRouteIndexers(ctx context.Context, mgr manager.Manager) error {

	err := mgr.GetFieldIndexer().IndexField(ctx, &gatewayapiv1.HTTPRoute{}, gatewayHTTPRouteIndex, func(rawObj client.Object) []string {
		var gateways []string
		httpRoute := rawObj.(*gatewayapiv1.HTTPRoute)
		for _, parentRefs := range httpRoute.Spec.ParentRefs {
			if parentRefs.Kind == nil || string(*parentRefs.Kind) == KindGateway {
				namespace := httpRoute.Namespace
				if parentRefs.Namespace != nil {
					namespace = string(*parentRefs.Namespace)
				}

				gateways = append(gateways, types.NamespacedName{
					Namespace: namespace,
					Name:      string(parentRefs.Name),
				}.String())
			}
		}

		return gateways
	})
	if err != nil {
		return err
	}

	return nil
}
