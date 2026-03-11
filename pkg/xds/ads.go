package xds

import (
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"time"
)

type DiscoveryStream = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer
type DeltaDiscoveryStream = discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer

func (s *DiscoveryServer) StreamAggregatedResources(stream DiscoveryStream) error {
	return s.Stream(stream)
}

func (s *DiscoveryServer) Stream(stream DiscoveryStream) error {

	if !s.IsServerReady() {
		return status.Error(codes.Unavailable, "server is not ready to serve discovery information")
	}

	ctx := stream.Context()
	peerAddr := "0.0.0.0"
	if peerInfo, ok := peer.FromContext(ctx); ok {
		peerAddr = peerInfo.Addr.String()
	}

	// grpc RequestRateLimit 请求限速
	if err := s.WaitForRequestLimit(stream.Context()); err != nil {
		klog.Warningf("ADS: %q exceeded rate limit: %v", peerAddr, err)
		return status.Errorf(codes.ResourceExhausted, "request rate limit exceeded: %v", err)
	}

	con := newConnectionContext(peerAddr, stream)
	//con.ids = ids
	con.s = s
	return Stream(con)
}

// Compute and send the new configuration for a connection.
func (s *DiscoveryServer) pushConnection(con *Connection, pushEv *Event) error {
	pushRequest := pushEv.pushRequest

	//if pushRequest.Full {
	//	// Update Proxy with current information.
	//	s.computeProxyState(con.proxy, pushRequest)
	//}

	pushRequest, needsPush := s.ProxyNeedsPush(con.proxy, pushRequest)
	if !needsPush {
		klog.Warningf("Skipping push to %v, no updates required", con.ID())
		return nil
	}

	// Send pushes to all generators
	// Each Generator is responsible for determining if the push event requires a push
	wrl := con.watchedResourcesByOrder()
	for _, w := range wrl {
		if err := s.pushXds(con, w, pushRequest); err != nil {
			return err
		}
	}
	//proxiesConvergeDelay.Record(time.Since(pushRequest.Start).Seconds())
	return nil
}
