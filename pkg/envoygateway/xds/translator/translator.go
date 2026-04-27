package translator

import (
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/xds/resource"
)

type Translator struct {

}

func (translator *Translator) Translate(xdsIR *ir.Xds) (*resource.ResourceVersionTable, error) {


	resources := new(resource.ResourceVersionTable)


	translator.processHTTPListenerXdsTranslation(resources)
}

