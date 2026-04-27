package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimev3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretv3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"

	listenerConfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	coreConfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routeConfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	clusterConfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointConfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	testv3 "github.com/envoyproxy/go-control-plane/pkg/test/v3"

	httpConnectionManagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	httpRouterv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"k8s.io/klog/v2"
	"net"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

const (
	ClusterName = "dynamic_cluster_example1"

	UpstreamAddr1 = "180.101.49.44" // "www.baidu.com" INFO: 必须是 IP，不能是域名，应该是哪里配置有问题。
	UpstreamAddr2 = "106.63.15.112" // www.sina.cn
)

var (
	debug  bool
	port   uint
	nodeID string
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Enable xDS server debug logging")

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 18000, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "test-id", "Node ID")
}

func main() {
	flag.Parse()

	ctx := ctrl.SetupSignalHandler()
	cache := cachev3.NewSnapshotCache(false, &cachev3.IDHash{}, nil)

	go GenerateNewSnapshot(ctx, cache)

	callbacks := &testv3.Callbacks{
		Debug: debug,
	}
	server := serverv3.NewServer(ctx, cache, callbacks)
	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(1000),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 5 * time.Second,
		}),
	}
	grpcServer := grpc.NewServer(grpcOptions...)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		klog.Fatalf("failed to listen error: %v", err)
	}

	// register xds server service
	discoveryv3.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	secretv3.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	clusterv3.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	endpointv3.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	listenerv3.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	routev3.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	runtimev3.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)

	go func() {
		klog.Infof("xDS gRPC server listening on %s", lis.Addr().String())
		if err = grpcServer.Serve(lis); err != nil {
			klog.Fatalf("xDS gRPC server error: %v", err)
		}
	}()

	<-ctx.Done()
	cache.ClearSnapshot(nodeID)
	grpcServer.Stop()
}

func GenerateNewSnapshot(ctx context.Context, cache cachev3.SnapshotCache) {
	upstreamAddrs := []string{UpstreamAddr1, UpstreamAddr2}
	for i := 0; i < len(upstreamAddrs); i++ {
		// INFO: 模拟做一些业务逻辑
		time.Sleep(time.Second * 10)

		upstreamAddr := upstreamAddrs[i]
		version := fmt.Sprintf("%d", i+1)
		snapshot, err := cachev3.NewSnapshot(version, map[resource.Type][]types.Resource{
			resource.ListenerType: {
				makeListener(),
			},
			resource.RouteType: {
				makeRoute(upstreamAddr),
			},
			resource.ClusterType: {
				makeCluster(upstreamAddr),
			},
		})
		if err != nil {
			klog.Fatalf("NewSnapshot err: %v", err)
		}
		if err = snapshot.Consistent(); err != nil {
			klog.Fatalf("snapshot.Consistent() error: %v", err)
		}

		klog.Infof("SetSnapshot nodeID: %s, version: %s", nodeID, version)
		err = cache.SetSnapshot(ctx, nodeID, snapshot)
		if err != nil {
			klog.Fatalf("SetSnapshot error: %v", err)
		}

		// INFO: 模拟做一些业务逻辑
		time.Sleep(time.Second * 60)
	}
}

/**
  listeners:
  - name: listener_0
    address:
      socket_address:
        address: 0.0.0.0
        port_value: 10000
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          access_log:
          - name: envoy.access_loggers.stdout
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
          http_filters:
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
          route_config:
            name: local_route
            virtual_hosts:
            - name: local_service
              domains: ["*"]
              routes:
              - match:
                  prefix: "/"
                route:
                  host_rewrite_literal: www.baidu.com
                  cluster: service_envoyproxy_io
*/
func makeListener() *listenerConfigv3.Listener {
	routerConfig, _ := anypb.New(&httpRouterv3.Router{})
	configSource := &coreConfigv3.ConfigSource{
		ConfigSourceSpecifier: &coreConfigv3.ConfigSource_ApiConfigSource{
			ApiConfigSource: &coreConfigv3.ApiConfigSource{
				ApiType:             coreConfigv3.ApiConfigSource_GRPC,
				TransportApiVersion: resource.DefaultAPIVersion,
				ClusterNames:        nil,
				GrpcServices: []*coreConfigv3.GrpcService{
					{
						TargetSpecifier: &coreConfigv3.GrpcService_EnvoyGrpc_{
							EnvoyGrpc: &coreConfigv3.GrpcService_EnvoyGrpc{
								ClusterName: "xds_cluster",
							},
						},
					},
				},
				SetNodeOnFirstMessageOnly: true,
			},
		},
		ResourceApiVersion: resource.DefaultAPIVersion,
	}
	// HTTP filter configuration
	httpConnectionManager := &httpConnectionManagerv3.HttpConnectionManager{
		CodecType:  httpConnectionManagerv3.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &httpConnectionManagerv3.HttpConnectionManager_Rds{
			Rds: &httpConnectionManagerv3.Rds{
				ConfigSource:    configSource,
				RouteConfigName: "local_route",
			},
		},
		HttpFilters: []*httpConnectionManagerv3.HttpFilter{
			{
				Name: "http-router",
				ConfigType: &httpConnectionManagerv3.HttpFilter_TypedConfig{
					TypedConfig: routerConfig,
				},
			},
		},
	}
	pbHttpConnectionManager, err := anypb.New(httpConnectionManager)
	if err != nil {
		panic(err)
	}

	return &listenerConfigv3.Listener{
		Name: "listener_0",
		Address: &coreConfigv3.Address{
			Address: &coreConfigv3.Address_SocketAddress{
				SocketAddress: &coreConfigv3.SocketAddress{
					Protocol: coreConfigv3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &coreConfigv3.SocketAddress_PortValue{
						PortValue: 10000,
					},
				},
			},
		},
		FilterChains: []*listenerConfigv3.FilterChain{
			{
				Filters: []*listenerConfigv3.Filter{
					{
						Name: "http-connection-manager",
						ConfigType: &listenerConfigv3.Filter_TypedConfig{
							TypedConfig: pbHttpConnectionManager,
						},
					},
				},
			},
		},
	}
}

/**
  route_config:
  name: local_route
  virtual_hosts:
    - name: local_service
      domains:
        - "*"
      routes:
        - match:
            prefix: "/"
          route:
            host_rewrite_literal: www.baidu.com
            cluster: example_proxy_cluster
*/
func makeRoute(upstreamAddr string) *routeConfigv3.RouteConfiguration {
	return &routeConfigv3.RouteConfiguration{
		Name: "local_route",
		VirtualHosts: []*routeConfigv3.VirtualHost{
			{
				Name:    "local_service",
				Domains: []string{"*"},
				Routes: []*routeConfigv3.Route{
					{
						Match: &routeConfigv3.RouteMatch{
							PathSpecifier: &routeConfigv3.RouteMatch_Prefix{
								Prefix: "/",
							},
						},
						Action: &routeConfigv3.Route_Route{
							Route: &routeConfigv3.RouteAction{
								ClusterSpecifier: &routeConfigv3.RouteAction_Cluster{
									Cluster: ClusterName,
								},
								HostRewriteSpecifier: &routeConfigv3.RouteAction_HostRewriteLiteral{
									HostRewriteLiteral: upstreamAddr,
								},
							},
						},
					},
				},
			},
		},
	}
}

func makeCluster(upstreamAddr string) *clusterConfigv3.Cluster {
	return &clusterConfigv3.Cluster{
		Name:                 ClusterName,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &clusterConfigv3.Cluster_Type{Type: clusterConfigv3.Cluster_LOGICAL_DNS},
		LbPolicy:             clusterConfigv3.Cluster_ROUND_ROBIN,
		DnsLookupFamily:      clusterConfigv3.Cluster_V4_ONLY,
		LoadAssignment: &endpointConfigv3.ClusterLoadAssignment{
			ClusterName: ClusterName,
			Endpoints: []*endpointConfigv3.LocalityLbEndpoints{
				{
					LbEndpoints: []*endpointConfigv3.LbEndpoint{
						{
							HostIdentifier: &endpointConfigv3.LbEndpoint_Endpoint{
								Endpoint: &endpointConfigv3.Endpoint{
									Address: &coreConfigv3.Address{
										Address: &coreConfigv3.Address_SocketAddress{
											SocketAddress: &coreConfigv3.SocketAddress{
												Address:  upstreamAddr,
												Protocol: coreConfigv3.SocketAddress_TCP,
												PortSpecifier: &coreConfigv3.SocketAddress_PortValue{
													PortValue: 80,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
