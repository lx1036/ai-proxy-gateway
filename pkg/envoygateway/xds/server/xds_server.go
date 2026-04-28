package server

import (
	"context"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimev3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretv3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/message"
	"github.com/lx1036/gateway/pkg/envoygateway/xds/cache"
	"github.com/lx1036/gateway/pkg/envoygateway/xds/translator"
	"github.com/telepresenceio/watchable"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"k8s.io/klog/v2"
	"net"
	"time"

	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

type XdsServer struct {
	grpcServer *grpc.Server

	cache cache.SnapshotCacheWithCallbacks

	XdsIR *message.XdsIR
}

func NewXdsServer(ctx context.Context, xdsIR *message.XdsIR) *XdsServer {

	xdsServer := &XdsServer{
		cache: cache.NewSnapshotCache(true),
		XdsIR: xdsIR,
	}

	grpcOpts := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(1000),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 5 * time.Second,
		}),
	}
	grpcServer := grpc.NewServer(grpcOpts...)

	srv := serverv3.NewServer(ctx, xdsServer.cache, nil)
	// srv := serverv3.NewServer(ctx, xdsServer.cache, nil) // TODO: 回头再实现 serverv3.Callbacks 接口
	discoveryv3.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)
	secretv3.RegisterSecretDiscoveryServiceServer(grpcServer, srv)
	clusterv3.RegisterClusterDiscoveryServiceServer(grpcServer, srv)
	endpointv3.RegisterEndpointDiscoveryServiceServer(grpcServer, srv)
	listenerv3.RegisterListenerDiscoveryServiceServer(grpcServer, srv)
	routev3.RegisterRouteDiscoveryServiceServer(grpcServer, srv)
	runtimev3.RegisterRuntimeDiscoveryServiceServer(grpcServer, srv)

	xdsServer.grpcServer = grpcServer

	return xdsServer
}

func (xdsServer *XdsServer) Start(ctx context.Context) error {

	go xdsServer.startXdsServer(ctx)

	// INFO: subscribe XdsIR message
	sub := xdsServer.XdsIR.Subscribe(ctx)
	go message.HandleSubscription(sub, func(update watchable.Update[string, *ir.Xds]) {

		key := update.Key
		xdsIR := update.Value

		if update.Delete { // Delete

			if err := xdsServer.cache.SetSnapshotCache(ctx, key, nil); err != nil {
				klog.Errorf("cache SetSnapshotCache error: %v", err)
				return
			}

		} else { // Create or Update

			t := translator.Translator{}
			result, err := t.Translate(xdsIR)
			if err != nil {
				klog.Errorf("translator Translate error: %v", err)
				return
			}

			if result.XdsResources != nil {

				if err := xdsServer.cache.SetSnapshotCache(ctx, key, result.XdsResources); err != nil {
					klog.Errorf("cache SetSnapshotCache error: %v", err)
					return
				}
			}

		}

	})

	return nil
}

func (xdsServer *XdsServer) startXdsServer(ctx context.Context) {
	l, err := net.Listen("tcp", "0.0.0.0:18000")
	if err != nil {
		klog.Errorf("xds server start failed: %v", err)
		return
	}

	go func() {
		<-ctx.Done()
		klog.Infof("force grpc server shutting down")
		xdsServer.grpcServer.Stop()
	}()

	if err = xdsServer.grpcServer.Serve(l); err != nil {
		klog.Errorf("xds server start failed: %v", err)
	}
}
