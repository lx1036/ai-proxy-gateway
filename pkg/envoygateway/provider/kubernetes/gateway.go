package kubernetes

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/envoygateway/message"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (gatewayAPIReconciler *GatewayAPIReconciler) processGateways(ctx context.Context, gatewayClass *gatewayapiv1.GatewayClass,
	resourceMap *resourceMappings, resources *message.GatewayAPIResource) error {

	var gatewayList gatewayapiv1.GatewayList
	err := gatewayAPIReconciler.mgr.GetClient().List(ctx, &gatewayList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.gatewayClassName", gatewayClass.Name),
	})
	if err != nil {
		return fmt.Errorf("list gateway error: %v", err)
	}

	for i := range gatewayList.Items {
		gateway := gatewayList.Items[i]

		klog.Infof("")

		resourceMap.allAssociatedNamespaces.Insert(gateway.Namespace)

		err = gatewayAPIReconciler.processHTTPRoutes(ctx, gateway, resourceMap, resources)


		key := types.NamespacedName{
			Namespace: gateway.Namespace,
			Name:      gateway.Name,
		}.String()
		if !resourceMap.allAssociatedGateways.Has(key) {
			resourceMap.allAssociatedGateways.Insert(key)
			resources.Gateways = append(resources.Gateways, &gateway)
		}


	}

	return nil
}
