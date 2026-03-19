package router

import (
	configPb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
)

func (s *Server) HandleRequestHeaders(request *extProcPb.ProcessingRequest) *extProcPb.ProcessingResponse {
	//requestHeaders := request.Request.(*extProcPb.ProcessingRequest_RequestHeaders)

	//requestHeaders.RequestHeaders.Headers.Headers

	headers := []*configPb.HeaderValueOption{
		&configPb.HeaderValueOption{
			Header: &configPb.HeaderValue{
				Key: HeaderWentIntoReqHeaders,
				//Value: "true",
				RawValue: []byte("true"),
			},
		},
	}

	return &extProcPb.ProcessingResponse{
		Response: &extProcPb.ProcessingResponse_RequestHeaders{
			RequestHeaders: &extProcPb.HeadersResponse{
				Response: &extProcPb.CommonResponse{
					HeaderMutation: &extProcPb.HeaderMutation{
						SetHeaders: headers,
					},
					ClearRouteCache: true,
				},
			},
		},
		DynamicMetadata:        nil,
		ModeOverride:           nil,
		RequestDrain:           false,
		OverrideMessageTimeout: nil,
	}

}
