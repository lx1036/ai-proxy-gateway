package extensionserver

import (
	"context"
	"fmt"

	egextension "github.com/envoyproxy/gateway/proto/extension"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

/**
 * Purpose: Modify individual routes
 * Execution Context: Per-route, only for routes with extension filters
 * Resources Modified: envoy.config.route.v3.Route
 *
 */

/**
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: inference-pool-with-httproute
  namespace: envoy-gateway-system
spec:
  parentRefs:
    - group: gateway.networking.k8s.io
      kind: Gateway
      name: inference-pool-with-httproute
      namespace: envoy-gateway-system
  rules:
    - backendRefs:
        # InferencePool 有且只能有一个
        - group: inference.networking.k8s.io
          kind: InferencePool
          name: envoy-ai-gateway-basic-testupstream
          namespace: envoy-gateway-system
          weight: 1
      matches:
        - path:
            type: PathPrefix
            value: /
      timeouts:
        request: 60s
*/

func (s *Server) PostRouteModify(_ context.Context, req *egextension.PostRouteModifyRequest) (*egextension.PostRouteModifyResponse, error) {
	if req.Route == nil {
		return nil, nil
	}

	// ExtensionResources 只能是 InferencePool resources

	// Check if we have backend extension resources (InferencePool resources).
	if req.PostRouteContext == nil || len(req.PostRouteContext.ExtensionResources) == 0 {
		// No backend extension resources, skip.
		return &egextension.PostRouteModifyResponse{Route: req.Route}, nil
	}

	// InferencePool 有且只能有一个
	inferencePools := s.constructInferencePoolsFrom(req.PostRouteContext.ExtensionResources)
	if len(inferencePools) == 0 {
		// No InferencePool resources, skip.
		return &egextension.PostRouteModifyResponse{Route: req.Route}, nil
	}

	if len(inferencePools) != 1 {
		return nil, fmt.Errorf("BUG: at most one inferencepool can be referenced per route rule but found %d", len(inferencePools))
	}

	// auto_host_rewrite: false

	// Disable auto host rewrite to prevent Envoy from overriding the host header
	// set by the endpoint picker. The endpoint picker sets the destination via
	// x-gateway-destination-endpoint header and we need to preserve the original
	// host for proper routing to the selected endpoint.
	req.Route.GetRoute().HostRewriteSpecifier = &routev3.RouteAction_AutoHostRewrite{
		AutoHostRewrite: wrapperspb.Bool(false),
	}
	if req.Route.TypedPerFilterConfig == nil {
		req.Route.TypedPerFilterConfig = make(map[string]*anypb.Any)
	}
	/**
	  metadata:
	    filter_metadata:
	      aigateway.envoy.io:
	        per_route_rule_inference_pool: envoy-gateway-system/envoy-ai-gateway-basic-testupstream/envoy-ai-gateway-basic-testupstream-epp/9002/duplex/false
	*/
	buildEPPMetadataForRoute(req.Route, inferencePools[0])

	return &egextension.PostRouteModifyResponse{Route: req.Route}, nil
}
