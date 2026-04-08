package server

import (
	"strings"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
)

const (
	RequestIdHeaderKey = "x-request-id"
)

func (s *ExtProcServer) HandleRequestHeaders(reqCtx *RequestContext, req *extProcPb.ProcessingRequest_RequestHeaders) error {
	reqCtx.RequestReceivedTimestamp = time.Now()

	// INFO: 这里考虑了 GET 请求
	// an EoS in the request headers means this request has no body or trailers.
	if req.RequestHeaders.EndOfStream {
		// currently any request without a body will be routed to a random upstream pod.
		// TODO: 待完善
	}

	for _, header := range req.RequestHeaders.Headers.Headers {
		reqCtx.Request.Headers[header.Key] = GetHeaderValue(header)
	}

	return nil
}

func ExtractHeadersValue(req *extProcPb.ProcessingRequest_RequestHeaders, headerKey string) string {
	headerKeyInLower := strings.ToLower(headerKey)
	if req != nil && req.RequestHeaders != nil && req.RequestHeaders.Headers != nil {
		for _, headerKv := range req.RequestHeaders.Headers.Headers {
			if strings.ToLower(headerKv.Key) == headerKeyInLower {
				return GetHeaderValue(headerKv)
			}
		}
	}
	return ""
}

func GetHeaderValue(header *corev3.HeaderValue) string {
	if len(header.RawValue) > 0 {
		return string(header.RawValue)
	}
	return header.Value
}
