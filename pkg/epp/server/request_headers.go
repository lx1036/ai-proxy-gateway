package server

import (
	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"time"
)

func (s *ExtProcServer) HandleRequestHeaders(reqCtx *RequestContext, req *extProcPb.ProcessingRequest_RequestHeaders) error {
	reqCtx.RequestReceivedTimestamp = time.Now()

	// INFO: 这里考虑了 GET
	// an EoS in the request headers means this request has no body or trailers.
	if req.RequestHeaders.EndOfStream {
		// We will route this request to a random pod as this is assumed to just be a GET
		// More context: https://github.com/kubernetes-sigs/gateway-api-inference-extension/pull/526
		// The above PR will address endpoint admission, but currently any request without a body will be
		// routed to a random upstream pod.

		// TODO: 待完善
	}

	return nil
}
