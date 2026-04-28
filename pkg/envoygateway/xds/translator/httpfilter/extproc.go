package httpfilter

import (
	"fmt"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	extprocv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_proc/v3"
	httpConnectionManagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/xds/resource"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"time"
)

func init() {
	RegisterHTTPFilter(&extProc{})
}

type extProc struct{}

func (e *extProc) patchHCM(mgr *httpConnectionManagerv3.HttpConnectionManager, irListener *ir.HTTPListener) error {
	if mgr == nil {
		return fmt.Errorf("hcm is nil")
	}

	if irListener == nil {
		return fmt.Errorf("ir listener is nil")
	}

	for _, route := range irListener.Routes {
		if !routeContainsExtProc(route) {
			continue
		}

		for i := range route.EnvoyExtensions.ExtProcs {
			extProcVal := &route.EnvoyExtensions.ExtProcs[i]
			if HttpConnectionManagerContainFilter(mgr, extProcFilterName(extProcVal)) {
				continue
			}

			filter, err := buildHTTPConnectionManagerExtProcFilter(extProcVal)

			mgr.HttpFilters = append(mgr.HttpFilters, filter)
		}

	}

}

func (e *extProc) patchRoute(route *routev3.Route, irRoute *ir.HTTPRoute, httpListener *ir.HTTPListener) error {
	//TODO implement me
	panic("implement me")
}

func (e *extProc) patchResources(tCtx *resource.ResourceVersionTable, routes []*ir.HTTPRoute) error {
	//TODO implement me
	panic("implement me")
}

func routeContainsExtProc(irRoute *ir.HTTPRoute) bool {
	if irRoute == nil {
		return false
	}

	return irRoute.EnvoyExtensions != nil && len(irRoute.EnvoyExtensions.ExtProcs) > 0
}

func buildHTTPConnectionManagerExtProcFilter(extProcVal ir.ExtProc) (*httpConnectionManagerv3.HttpFilter, error) {
	extAuthProto, err := extProcConfig(extProcVal)
	if err != nil {
		return nil, err
	}
	extAuthAny, err := anypb.New(extAuthProto)
	if err != nil {
		return nil, err
	}

	return &httpConnectionManagerv3.HttpFilter{
		Name: extProcFilterName(extProcVal),
		ConfigType: &httpConnectionManagerv3.HttpFilter_TypedConfig{
			TypedConfig: extAuthAny,
		},
	}, nil
}

func extProcConfig(extProc *ir.ExtProc) (*extprocv3.ExternalProcessor, error) {
	extProConfig := &extprocv3.ExternalProcessor{
		GrpcService: &corev3.GrpcService{
			TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
				EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
					ClusterName: extProc.Destination.Name,
					Authority:   extProc.Authority,
				},
			},
			Timeout: durationpb.New(time.Second * 10),
		},
		ProcessingMode: buildProcessingMode(extProc),
	}
}

func buildProcessingMode(extProc *ir.ExtProc) *extprocv3.ProcessingMode {
	processingMode := &extprocv3.ProcessingMode{
		RequestHeaderMode:   extprocv3.ProcessingMode_SKIP,
		ResponseHeaderMode:  extprocv3.ProcessingMode_SKIP,
		RequestBodyMode:     extprocv3.ProcessingMode_NONE,
		ResponseBodyMode:    extprocv3.ProcessingMode_NONE,
		RequestTrailerMode:  extprocv3.ProcessingMode_SKIP,
		ResponseTrailerMode: extprocv3.ProcessingMode_SKIP,
	}

	if extProc.RequestHeaderProcessing {
		processingMode.RequestHeaderMode = extprocv3.ProcessingMode_SEND
	}

	if extProc.RequestBodyProcessingMode != nil {
		processingMode.RequestBodyMode = translateExtProcBodyProcessingMode(extProc.RequestBodyProcessingMode)
		if processingMode.RequestBodyMode == extprocv3.ProcessingMode_FULL_DUPLEX_STREAMED {
			processingMode.RequestTrailerMode = extprocv3.ProcessingMode_SEND
		}
	}

	if extProc.ResponseHeaderProcessing {
		processingMode.ResponseHeaderMode = extprocv3.ProcessingMode_SEND
	}

	if extProc.ResponseBodyProcessingMode != nil {
		processingMode.ResponseBodyMode = translateExtProcBodyProcessingMode(extProc.ResponseBodyProcessingMode)
		if processingMode.ResponseBodyMode == extprocv3.ProcessingMode_FULL_DUPLEX_STREAMED {
			processingMode.ResponseTrailerMode = extprocv3.ProcessingMode_SEND
		}
	}

	return processingMode
}

func translateExtProcBodyProcessingMode(mode *ir.ExtProcBodyProcessingMode) extprocv3.ProcessingMode_BodySendMode {
	lookup := map[ir.ExtProcBodyProcessingMode]extprocv3.ProcessingMode_BodySendMode{
		ir.ExtProcBodyBuffered:           extprocv3.ProcessingMode_BUFFERED,
		ir.ExtProcBodyBufferedPartial:    extprocv3.ProcessingMode_BUFFERED_PARTIAL,
		ir.ExtProcBodyStreamed:           extprocv3.ProcessingMode_STREAMED,
		ir.ExtProcBodyFullDuplexStreamed: extprocv3.ProcessingMode_FULL_DUPLEX_STREAMED,
	}
	if r, found := lookup[*mode]; found {
		return r
	}
	return extprocv3.ProcessingMode_NONE
}
