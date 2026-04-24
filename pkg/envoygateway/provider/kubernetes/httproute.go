package kubernetes

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/envoygateway/gatewayapi/resource"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (gatewayAPIReconciler *GatewayAPIReconciler) processHTTPRoutes(ctx context.Context, gateway gatewayapiv1.Gateway, resources *resource.Resources) error {

	// kubectl get httproute -A --field-selector spec.parentRefs=envoy-gateway-system/envoy-gateway
	var httpRouteList gatewayapiv1.HTTPRouteList
	err := gatewayAPIReconciler.mgr.GetClient().List(ctx, &httpRouteList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(gatewayHTTPRouteIndex, types.NamespacedName{
			Namespace: gateway.Namespace,
			Name:      gateway.Name,
		}.String()),
	})
	if err != nil {
		return fmt.Errorf("list httproute error: %v", err)
	}

	klog.Infof("list %d HTTPRoute for gateway %s/%s", len(httpRouteList.Items), gateway.Namespace, gateway.Name)

	for i := range httpRouteList.Items {
		httpRoute := &httpRouteList.Items[i]
		gatewayAPIReconciler.processHTTPRoute(ctx, httpRoute, resources)
	}

	return nil
}

func (gatewayAPIReconciler *GatewayAPIReconciler) processHTTPRoute(ctx context.Context, httpRoute *gatewayapiv1.HTTPRoute,
	resources *resource.Resources) {

	for _, rule := range httpRoute.Spec.Rules {

		for _, backendRefs := range rule.BackendRefs {

			gatewayAPIReconciler.processBackendRef(ctx, backendRefs)

			for _, filter := range backendRefs.Filters {
				gatewayAPIReconciler.processHTTPRouteFilter(ctx, filter)
			}

		}

		for _, filter := range rule.Filters {
			gatewayAPIReconciler.processHTTPRouteFilter(ctx, filter)
		}
	}

	resources.HTTPRoutes = append(resources.HTTPRoutes, httpRoute)

}

func (gatewayAPIReconciler *GatewayAPIReconciler) processBackendRef(ctx context.Context, backendRef gatewayapiv1.HTTPBackendRef) {

	// TODO: ReferenceGrant
}

func (gatewayAPIReconciler *GatewayAPIReconciler) processHTTPRouteFilter(ctx context.Context, filter gatewayapiv1.HTTPRouteFilter) {
	switch filter.Type {
	case gatewayapiv1.HTTPRouteFilterRequestMirror:
		//gatewayAPIReconciler.processHTTPRouteFilterRequestMirror(ctx, filter.RequestMirror)
	case gatewayapiv1.HTTPRouteFilterRequestRedirect:
		//gatewayAPIReconciler.processHTTPRouteFilterRequestRedirect(ctx, filter.RequestRedirect)
	case gatewayapiv1.HTTPRouteFilterURLRewrite:
		//gatewayAPIReconciler.processHTTPRouteFilterURLRewrite()
	default:
		//gatewayAPIReconciler.processHTTPRouteFilterExtensionRef(ctx, filter.ExtensionRef)
	}
}
