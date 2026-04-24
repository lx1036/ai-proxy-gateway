package kubernetes

import (
	"context"
	"fmt"
	"github.com/lx1036/gateway/pkg/envoygateway/gatewayapi/resource"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (gatewayAPIReconciler *GatewayAPIReconciler) processGateways(ctx context.Context, gatewayClass *gatewayapiv1.GatewayClass, resources *resource.Resources) error {

	var gatewayList gatewayapiv1.GatewayList
	err := gatewayAPIReconciler.mgr.GetClient().List(ctx, &gatewayList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.gatewayClassName", gatewayClass.Name),
	})
	if err != nil {
		return fmt.Errorf("list gateway error: %v", err)
	}

	for i := range gatewayList.Items {
		gateway := gatewayList.Items[i]


		err = gatewayAPIReconciler.processHTTPRoutes(ctx, gateway, resources)




		resources.Gateways = append(resources.Gateways, &gateway)

	}

	return nil
}
