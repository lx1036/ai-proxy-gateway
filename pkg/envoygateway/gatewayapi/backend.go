package gatewayapi

import (
	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/scheme"
	"k8s.io/utils/ptr"
	"strconv"
)

func (translator *Translator) translateExtServiceBackendRefs(extProc envoygatewayv1alpha1.ExtProc) {

	ds := make([]*ir.DestinationSetting, 0, len(extProc.BackendRefs))
	for _, backendRef := range extProc.BackendRefs {

		extServiceDest, err := translator.processExtServiceDestination(backendRef)

		ds = append(ds, extServiceDest)

	}

	routeDestination := &ir.RouteDestination{}

}

func (translator *Translator) processExtServiceDestination(backendRef envoygatewayv1alpha1.BackendRef) (*ir.DestinationSetting, error) {
	var (
		destinationSetting *ir.DestinationSetting
	)

	switch KindDerefOr(backendRef.Kind, scheme.KindService) {
	case scheme.KindService:
		destinationSetting, err = translator.processServiceDestinationSetting()

	case scheme.KindServiceImport:
		// TODO

	case envoygatewayv1alpha1.KindBackend:
	}

	return destinationSetting, nil
}

func (translator *Translator) processServiceDestinationSetting() (*ir.DestinationSetting, error) {

	return &ir.DestinationSetting{
		Name:        name,
		Protocol:    protocol,
		Endpoints:   endpoints,
		AddressType: addrType,
		//PreferLocal: processPreferLocalZone(service),
		Metadata: buildResourceMetadata(service, ptr.To(gwapiv1.SectionName(strconv.Itoa(int(*backendRef.Port))))),
	}, nil

}
