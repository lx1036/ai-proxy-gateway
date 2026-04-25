package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func GatewayNotAccepted(gw *gatewayapiv1.Gateway) bool {
	for _, c := range gw.Status.Conditions {
		if c.Type == string(gatewayapiv1.GatewayConditionAccepted) && c.Status == metav1.ConditionFalse {
			return true
		}
	}
	return false
}
