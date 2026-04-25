package gatewayapi

import (
	"github.com/lx1036/gateway/pkg/envoygateway/gatewayapi/resource"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type HTTPRouteContext struct {
	*gatewayapiv1.HTTPRoute
}


func (translator *Translator) ProcessHTTPRoutes(resources *resource.Resources) []*HTTPRouteContext {
	relevantHTTPRoutes := make([]*HTTPRouteContext, 0, len(resources.HTTPRoutes))

	for _, httpRoute := range resources.HTTPRoutes {

		httpRouteContext := &HTTPRouteContext{
			HTTPRoute: httpRoute,
		}

		relevantHTTPRoutes = append(relevantHTTPRoutes, httpRouteContext)
	}


	return relevantHTTPRoutes
}
