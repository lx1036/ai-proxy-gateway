package extensionserver

import (
	"context"
	"fmt"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"k8s.io/klog/v2"
	"strconv"
	"strings"
	"time"

	egextension "github.com/envoyproxy/gateway/proto/extension"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	extprocv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_proc/v3"
	httpconnectionmanagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	upstreamsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gwaiev1 "sigs.k8s.io/gateway-api-inference-extension/api/v1"
)

/**
 * Purpose: Global modification of all xDS resources
 * Execution Context: Once per translation cycle
 * Resources Modified: All clusters, secrets, listeners, and routes
 *
 */

// PostTranslateModify allows an extension to modify the clusters and secrets in the xDS config
// after the initial translation is complete. This method is responsible for:
//
// 1. Modifying existing clusters (e.g., adding metadata, adjusting configurations)
// 2. Adding additional clusters needed for InferencePool support (EPP clusters)
// 3. Ensuring the AI Gateway external processor UDS cluster exists TODO:暂且不做
//
// For InferencePool support, this method creates additional STRICT_DNS clusters that
// connect to the endpoint picker services specified in InferencePool resources.
func (s *Server) PostTranslateModify(_ context.Context, req *egextension.PostTranslateModifyRequest) (*egextension.PostTranslateModifyResponse, error) {
	// Add external processor clusters for InferencePool backends.
	// These clusters connect to the endpoint picker services (EPP) specified in InferencePool resources.
	cs, err := buildClustersForInferencePoolEndpointPickers(req.Clusters)
	if err != nil {
		return nil, fmt.Errorf("failed to build clusters for InferencePool endpoint pickers: %w", err)
	}
	req.Clusters = append(req.Clusters, cs...)

	// Modify listeners and routes to support InferencePool backends.
	if err = s.maybeModifyListenerAndRoutes(req.Listeners, req.Routes); err != nil {
		return nil, fmt.Errorf("failed to modify listeners and routes for InferencePool support: %w", err)
	}

	return &egextension.PostTranslateModifyResponse{
		Clusters:  req.Clusters,
		Secrets:   req.Secrets,
		Listeners: req.Listeners,
		Routes:    req.Routes,
	}, nil
}

// buildClustersForInferencePoolEndpointPickers builds and returns a "STRICT_DNS" cluster
// for each InferencePool's endpoint picker service.
func buildClustersForInferencePoolEndpointPickers(clusters []*clusterv3.Cluster) ([]*clusterv3.Cluster, error) {
	result := make([]*clusterv3.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		if pool := getInferencePoolByMetadata(cluster.Metadata); pool != nil {
			c, err := buildExtProcClusterForInferencePoolEndpointPicker(pool)
			if err != nil {
				return nil, err
			}
			result = append(result, c)
		}
	}
	return result, nil
}

// maybeModifyListenerAndRoutes modifies listeners and routes to support InferencePool backends.
// This function performs the following operations:
// 1. Identifies listeners and routes that use InferencePool backends
// 2. Adds endpoint picker (EPP) external processor filters to relevant listeners
// 3. Configures per-route filters to disable EPP processing for non-InferencePool routes
// This ensures that only routes targeting InferencePool backends go through the endpoint picker.
func (s *Server) maybeModifyListenerAndRoutes(listeners []*listenerv3.Listener, routes []*routev3.RouteConfiguration) error {
	listenerNameToRouteNames := make(map[string][]string)
	listenerNameToListener := make(map[string]*listenerv3.Listener)
	for _, listener := range listeners {
		// Skips two special listeners used for stats and readiness probes created by Envoy Gateway.
		// https://github.com/envoyproxy/gateway/blob/e7c0e3430f20fd104454dc62f51b5352cde3b26a/internal/xds/bootstrap/bootstrap.yaml.tpl#L89
		// https://github.com/envoyproxy/gateway/blob/e7c0e3430f20fd104454dc62f51b5352cde3b26a/internal/xds/translator/listener_ready.go#L76
		//
		// This assumes that EG keeps the same naming convention, but even if not, the latest EG e2e test will catch any issue.
		//
		// The "normal" user-created listeners will have "${gateway obj namespace}-${gateway obj name}-..." format.
		if strings.HasPrefix(listener.Name, "envoy-gateway-proxy-stats-") || strings.HasPrefix(listener.Name, "envoy-gateway-proxy-ready-") {
			continue
		}
		listenerNameToRouteNames[listener.Name] = findListenerRouteConfigs(listener)
		listenerNameToListener[listener.Name] = listener
	}

	// inferencePoolRoutes builds a matrix of route configs and the inference pools they use.
	routeNameToRoute := make(map[string]*routev3.RouteConfiguration)
	routeNameToVHRouteNameToInferencePool := make(map[string]map[string]*gwaiev1.InferencePool)
	for _, routeCfg := range routes {
		routeNameToRoute[routeCfg.Name] = routeCfg
		for _, vh := range routeCfg.VirtualHosts {
			for _, route := range vh.Routes {
				if pool := getInferencePoolByMetadata(route.Metadata); pool != nil {
					if routeNameToVHRouteNameToInferencePool[routeCfg.Name] == nil {
						routeNameToVHRouteNameToInferencePool[routeCfg.Name] = make(map[string]*gwaiev1.InferencePool)
					}
					routeNameToVHRouteNameToInferencePool[routeCfg.Name][route.Name] = pool
				}
			}
		}
	}

	// listenerToInferencePools builds a matrix of listeners and the inference pools they use.
	listenerToInferencePools := make(map[string][]*gwaiev1.InferencePool)
	for listener, routeCfgNames := range listenerNameToRouteNames {
		for _, name := range routeCfgNames {
			if routeNameToRoute[name] == nil {
				continue
			}
			if routeNameToVHRouteNameToInferencePool[name] == nil {
				continue
			}
			for _, pool := range routeNameToVHRouteNameToInferencePool[name] {
				if listenerToInferencePools[listener] == nil {
					listenerToInferencePools[listener] = make([]*gwaiev1.InferencePool, 0)
				}
				listenerToInferencePools[listener] = append(listenerToInferencePools[listener], pool)
			}
		}
	}

	// patch the listeners, the route configs and the virtual hosts with inference pool filters.
	for listener, pools := range listenerToInferencePools {
		klog.InfoS("patching listener with inference pool filters", "listener", listener)
		s.patchListenerWithInferencePoolFilters(listenerNameToListener[listener], pools)
		routeCfgNames := listenerNameToRouteNames[listener]
		for _, routeCfgName := range routeCfgNames {
			routeCfg := routeNameToRoute[routeCfgName]
			if routeCfg == nil {
				continue
			}
			for _, vh := range routeCfg.VirtualHosts {
				klog.InfoS("patching virtual host with inference pool filters", "listener", listener, "virtual_host", vh.Name)
				if err := s.patchVirtualHostWithInferencePool(vh, pools); err != nil {
					return fmt.Errorf("failed to patch virtual host %s in route config %s: %w", vh.Name, routeCfg.Name, err)
				}
			}
		}
	}

	return nil
}

/**
添加一个 ext-proc HTTPFilter 配置:

  http_filters:
    - name: envoy.filters.http.ext_proc/endpointpicker/envoy-ai-gateway-basic-testupstream_envoy-gateway-system_ext_proc
      typed_config:
        '@type': type.googleapis.com/envoy.extensions.filters.http.ext_proc.v3.ExternalProcessor
        grpc_service:
          envoy_grpc:
            cluster_name: envoy.clusters.endpointpicker_envoy-ai-gateway-basic-testupstream_envoy-gateway-system_ext_proc
            authority: envoy-ai-gateway-basic-testupstream-epp.envoy-gateway-system.svc:9002
        processing_mode:
          request_header_mode: SEND
          response_header_mode: SEND
          request_body_mode: FULL_DUPLEX_STREAMED
          response_body_mode: FULL_DUPLEX_STREAMED
          request_trailer_mode: SEND
          response_trailer_mode: SEND
        message_timeout: 300s
*/
// patchListenerWithInferencePoolFilters adds the necessary HTTP filters to the listener to support InferencePool backends.
func (s *Server) patchListenerWithInferencePoolFilters(listener *listenerv3.Listener, inferencePools []*gwaiev1.InferencePool) {
	// First, get the filter chains from the listener.
	filterChains := listener.GetFilterChains()
	defaultFC := listener.DefaultFilterChain
	if defaultFC != nil {
		filterChains = append(filterChains, defaultFC)
	}
	// Go over all of the chains, and add the endpoint picker external processor filters.
	for _, currChain := range filterChains {
		httpConManager, hcmIndex, err := findHttpConnectionManager(currChain)
		if err != nil {
			klog.Errorf("failed to find an HCM in the current chain: %v", err)
			continue
		}
		var poolFilters []*httpconnectionmanagerv3.HttpFilter
		for _, pool := range inferencePools {
			_, baIndex, searchErr := searchInferencePoolInFilterChain(pool, httpConManager.HttpFilters)
			if searchErr != nil {
				klog.Errorf("failed to find an inference pool ext proc filter: %v", searchErr)
				continue
			}
			// 没有找到，则添加
			if baIndex == -1 {
				klog.Infof("adding inference pool ext proc filter", "pool", pool.Name)
				var eppExtProc *httpconnectionmanagerv3.HttpFilter
				eppExtProc, err = buildInferencePoolHTTPFilter(pool)
				if err != nil {
					klog.ErrorS(err, "failed to build inference pool ext proc filter", "pool", pool.Name)
					continue
				}
				poolFilters = append(poolFilters, eppExtProc)
			}
		}
		if len(poolFilters) != 0 {
			length := len(httpConManager.HttpFilters)
			router := httpConManager.HttpFilters[length-1]
			httpConManager.HttpFilters = httpConManager.HttpFilters[:length-1]
			httpConManager.HttpFilters = append(httpConManager.HttpFilters, poolFilters...)
			httpConManager.HttpFilters = append(httpConManager.HttpFilters, router)
		}

		// Write the updated HCM back to the filter chain.
		hcmAny, err := toAny(httpConManager)
		if err != nil {
			klog.Errorf("failed to marshal updated HCM to Any: %v", err)
			continue
		}
		currChain.Filters[hcmIndex].ConfigType = &listenerv3.Filter_TypedConfig{TypedConfig: hcmAny}
	}
}

// patchVirtualHostWithInferencePool adds the necessary per-route configuration to disable.
func (s *Server) patchVirtualHostWithInferencePool(vh *routev3.VirtualHost, inferencePools []*gwaiev1.InferencePool) error {
	inferenceMatrix := make(map[string]*gwaiev1.InferencePool)
	for _, pool := range inferencePools {
		inferenceMatrix[httpFilterNameForInferencePool(pool)] = pool
	}
	for _, route := range vh.Routes {
		override := &extprocv3.ExtProcPerRoute{
			Override: &extprocv3.ExtProcPerRoute_Disabled{
				Disabled: true,
			},
		}
		overrideAny, err := toAny(override)
		if err != nil {
			return fmt.Errorf("failed to marshal ExtProcPerRoute to Any: %w", err)
		}
		inferencePool := getInferencePoolByMetadata(route.Metadata)
		if inferencePool == nil {
			for key, pool := range inferenceMatrix {
				klog.InfoS("disabling inference pool filter", "route", route.Name, "filter", key, "pool", pool.Name)
				if route.TypedPerFilterConfig == nil {
					route.TypedPerFilterConfig = make(map[string]*anypb.Any)
				}
				route.TypedPerFilterConfig[key] = overrideAny
			}
		} else {
			for key, pool := range inferenceMatrix {
				if key != httpFilterNameForInferencePool(inferencePool) {
					klog.InfoS("disabling inference pool filter", "route", route.Name, "filter", key, "pool", pool.Name)
					if route.TypedPerFilterConfig == nil {
						route.TypedPerFilterConfig = make(map[string]*anypb.Any)
					}
					route.TypedPerFilterConfig[key] = overrideAny
				}
			}
		}
	}

	return nil
}

// Tries to find the inference pool ext proc filter in the provided chain.
func searchInferencePoolInFilterChain(pool *gwaiev1.InferencePool, chain []*httpconnectionmanagerv3.HttpFilter) (*extprocv3.ExternalProcessor, int, error) {
	for i, filter := range chain {
		if filter.Name == httpFilterNameForInferencePool(pool) {
			ep := new(extprocv3.ExternalProcessor)
			if err := filter.GetTypedConfig().UnmarshalTo(ep); err != nil {
				return nil, -1, err
			}
			return ep, i, nil
		}
	}
	return nil, -1, nil
}

// findListenerRouteConfigs extracts route configuration names from the listener's filter chains.
func findListenerRouteConfigs(listener *listenerv3.Listener) []string {
	var names []string
	// First, get the filter chains from the listener.
	for _, filterChain := range listener.FilterChains {
		httpConManager, _, err := findHttpConnectionManager(filterChain)
		if err != nil {
			continue // Skip this filter chain if it doesn't have an HTTP connection manager.
		}
		rds := httpConManager.GetRds()
		if rds == nil {
			continue // Skip if no route discovery service is configured.
		}
		if rds.RouteConfigName != "" {
			names = append(names, rds.RouteConfigName)
		}
	}

	httpConManager, _, err := findHttpConnectionManager(listener.DefaultFilterChain)
	if err != nil {
		return names // Return names collected so far, even if default filter chain has no HCM.
	}
	rds := httpConManager.GetRds()
	if rds == nil {
		return names // Return names collected so far, even if no RDS in default filter chain.
	}

	return append(names, rds.RouteConfigName) // Add default filter chain's route config name.
}

// Tries to find an HTTP connection manager in the provided filter chain.
func findHttpConnectionManager(filterChain *listenerv3.FilterChain) (*httpconnectionmanagerv3.HttpConnectionManager, int, error) {
	if filterChain == nil {
		return nil, -1, fmt.Errorf("filter chain is nil")
	}
	for filterIndex, filter := range filterChain.Filters {
		if filter.Name == wellknown.HTTPConnectionManager {
			hcm := new(httpconnectionmanagerv3.HttpConnectionManager)
			if err := filter.GetTypedConfig().UnmarshalTo(hcm); err != nil {
				return nil, -1, err
			}
			return hcm, filterIndex, nil
		}
	}
	return nil, -1, fmt.Errorf("unable to find HTTPConnectionManager in FilterChain: %s", filterChain.Name)
}

/*
*
buildEPPMetadataForCluster() 添加的 per_route_rule_inference_pool metadata，这里使用来组成一个 InferencePool 资源对象

	metadata:
	  filter_metadata:
	    aigateway.envoy.io:
	      per_route_rule_inference_pool: envoy-gateway-system/envoy-ai-gateway-basic-testupstream/envoy-ai-gateway-basic-testupstream-epp/9002/duplex/false
*/
func getInferencePoolByMetadata(meta *corev3.Metadata) *gwaiev1.InferencePool {
	var metadata string
	if meta != nil && meta.FilterMetadata != nil {
		m, ok := meta.FilterMetadata[InternalEndpointMetadataNamespace]
		if ok && m.Fields != nil {
			v, ok := m.Fields[internalMetadataInferencePoolKey]
			if ok {
				metadata = v.GetStringValue()
			}
		}
	}

	result := strings.Split(metadata, "/")
	if len(result) != 6 {
		return nil
	}
	ns := result[0]
	name := result[1]
	serviceName := result[2]
	port, err := strconv.ParseInt(result[3], 10, 32)
	if err != nil {
		return nil
	}
	processingBodyMode := result[4]
	allowModeOverride := result[5]
	return &gwaiev1.InferencePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Annotations: map[string]string{
				processingBodyModeAnnotation: processingBodyMode,
				allowModeOverrideAnnotation:  allowModeOverride,
			},
		},
		Spec: gwaiev1.InferencePoolSpec{
			EndpointPickerRef: gwaiev1.EndpointPickerRef{
				Name: gwaiev1.ObjectName(serviceName),
				Port: ptr.To(gwaiev1.Port{Number: gwaiev1.PortNumber(port)}),
			},
		},
	}
}

/**
  # buildExtProcClusterForInferencePoolEndpointPicker()
  - version_info: b0cb5060d3d5d8b19327e9e746f4a6ef505c61123beaf3ef600f03330f9764fa
    cluster:
      '@type': type.googleapis.com/envoy.config.cluster.v3.Cluster
      name: envoy.clusters.endpointpicker_envoy-ai-gateway-basic-testupstream_envoy-gateway-system_ext_proc
      type: STRICT_DNS
      connect_timeout: 10s
      lb_policy: LEAST_REQUEST
      transport_socket:
        name: envoy.transport_sockets.tls
        typed_config:
          '@type': type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
          common_tls_context:
            validation_context: {}
      load_assignment:
        cluster_name: envoy.clusters.endpointpicker_envoy-ai-gateway-basic-testupstream_envoy-gateway-system_ext_proc
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: envoy-ai-gateway-basic-testupstream-epp.envoy-gateway-system.svc
                      port_value: 9002
                health_status: HEALTHY
      typed_extension_protocol_options:
        envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
          '@type': type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
          explicit_http_config:
            http2_protocol_options: {}
    last_updated: "2026-03-18T11:50:39.497Z"
*/
// buildExtProcClusterForInferencePoolEndpointPicker builds and returns a "STRICT_DNS" cluster
// for connecting to the InferencePool's endpoint picker service.
func buildExtProcClusterForInferencePoolEndpointPicker(pool *gwaiev1.InferencePool) (*clusterv3.Cluster, error) {
	name := clusterNameForInferencePool(pool)
	anyTLS, err := toAny(&tlsv3.UpstreamTlsContext{
		CommonTlsContext: &tlsv3.CommonTlsContext{
			ValidationContextType: &tlsv3.CommonTlsContext_ValidationContext{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS context for InferencePool cluster %s: %w", name, err)
	}
	c := &clusterv3.Cluster{
		Name:           name,
		ConnectTimeout: durationpb.New(10 * time.Second),
		ClusterDiscoveryType: &clusterv3.Cluster_Type{
			Type: clusterv3.Cluster_STRICT_DNS,
		},
		LbPolicy: clusterv3.Cluster_LEAST_REQUEST,
		// Ensure Envoy accepts untrusted certificates.
		TransportSocket: &corev3.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &corev3.TransportSocket_TypedConfig{
				TypedConfig: anyTLS,
			},
		},
		LoadAssignment: &endpointv3.ClusterLoadAssignment{
			ClusterName: name,
			Endpoints: []*endpointv3.LocalityLbEndpoints{{
				LbEndpoints: []*endpointv3.LbEndpoint{{
					HealthStatus: corev3.HealthStatus_HEALTHY,
					HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
						Endpoint: &endpointv3.Endpoint{
							Address: &corev3.Address{
								Address: &corev3.Address_SocketAddress{
									SocketAddress: &corev3.SocketAddress{
										Address:  dnsNameForInferencePool(pool),
										Protocol: corev3.SocketAddress_TCP,
										PortSpecifier: &corev3.SocketAddress_PortValue{
											PortValue: portForInferencePool(pool),
										},
									},
								},
							},
						},
					},
				}},
			}},
		},
	}

	http2Opts := &upstreamsv3.HttpProtocolOptions{
		UpstreamProtocolOptions: &upstreamsv3.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &upstreamsv3.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &upstreamsv3.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{
					Http2ProtocolOptions: &corev3.Http2ProtocolOptions{},
				},
			},
		},
	}
	anyHTTP2, err := toAny(http2Opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP2 options for InferencePool cluster %s: %w", name, err)
	}
	c.TypedExtensionProtocolOptions = map[string]*anypb.Any{
		"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": anyHTTP2,
	}

	return c, nil
}

// toAny marshals the provided message to an Any message.
func toAny(msg proto.Message) (*anypb.Any, error) {
	b, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message to Any: %w", err)
	}
	const envoyAPIPrefix = "type.googleapis.com/"
	return &anypb.Any{
		TypeUrl: envoyAPIPrefix + string(msg.ProtoReflect().Descriptor().FullName()),
		Value:   b,
	}, nil
}

func dnsNameForInferencePool(pool *gwaiev1.InferencePool) string {
	ns := pool.GetNamespace()
	svc := pool.Spec.EndpointPickerRef.Name
	return fmt.Sprintf("%s.%s.svc", svc, ns)
}
