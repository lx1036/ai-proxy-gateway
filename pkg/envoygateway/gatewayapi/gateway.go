package gatewayapi

import (
	"github.com/lx1036/gateway/pkg/envoygateway/gatewayapi/resource"
	"github.com/lx1036/gateway/pkg/envoygateway/gatewayapi/status"
	"k8s.io/klog/v2"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayContext struct {
	*gatewayapiv1.Gateway
}

func (translator *Translator) GetRelevantGateways(resources *resource.Resources) (acceptedGateways, failedGateways []*GatewayContext) {

	for _, gateway := range resources.Gateways {
		gCtx := &GatewayContext{
			Gateway: gateway,
		}

		// Gateways that are not accepted by the controller because they reference an invalid EnvoyProxy.
		if status.GatewayNotAccepted(gCtx.Gateway) {
			failedGateways = append(failedGateways, gCtx)
			klog.Warningf("EnvoyProxy for Gateway %s/%s not found", gCtx.Gateway.Namespace, gCtx.Gateway.Name)
			continue
		}

		acceptedGateways = append(acceptedGateways, gCtx)
	}

	return acceptedGateways, failedGateways
}
