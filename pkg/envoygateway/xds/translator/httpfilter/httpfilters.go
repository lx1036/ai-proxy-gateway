package httpfilter

import (
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/xds/resource"

	httpConnectionManagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

)

type HTTPFilter interface {
	patchHCM(mgr *httpConnectionManagerv3.HttpConnectionManager, irListener *ir.HTTPListener) error


	patchRoute(route *routev3.Route, irRoute *ir.HTTPRoute, httpListener *ir.HTTPListener) error


	patchResources(tCtx *resource.ResourceVersionTable, routes []*ir.HTTPRoute) error

}


var HTTPFilters []HTTPFilter

func RegisterHTTPFilter(filter HTTPFilter) {
	HTTPFilters = append(HTTPFilters, filter)
}
