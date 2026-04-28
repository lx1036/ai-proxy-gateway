package translator

import (
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/xds/resource"
)

func (translator *Translator) addRouteToRouteConfig(resourceTable *resource.ResourceVersionTable, httpListener *ir.HTTPListener) {

	for _, httpRoute := range httpListener.Routes {

	}

}
