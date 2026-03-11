package xds

import (
	"context"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/lx1036/gateway/pkg/model"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"time"

	"go.uber.org/atomic"
)

// DiscoveryServer is GatewayController gRPC implementation for Envoy's xds APIs
type DiscoveryServer struct {

	// serverReady indicates caches have been synced up and server is ready to process requests.
	serverReady atomic.Bool
	// DiscoveryStartTime is the time since the binary started
	DiscoveryStartTime time.Time

	// RequestRateLimit limits the number of new XDS requests allowed. This helps prevent thundering hurd of incoming requests.
	RequestRateLimit *rate.Limiter

	// InboundUpdates describes the number of configuration updates the discovery server has received
	InboundUpdates *atomic.Int64
	// CommittedUpdates describes the number of configuration updates the discovery server has
	// received, process, and stored in the push context. If this number is less than InboundUpdates,
	// there are updates we have not yet processed.
	// Note: This does not mean that all proxies have received these configurations; it is strictly
	// the push context, which means that the next push to a proxy will receive this configuration.
	CommittedUpdates *atomic.Int64

	// Generators allow customizing the generated config, based on the client metadata.
	// Key is the generator type - will match the Generator metadata to set the per-connection
	// default generator, or the combination of Generator metadata and TypeUrl to select a
	// different generator for a type.
	// Normal istio clients use the default generator - will not be impacted by this.
	Generators map[string]XdsResourceGenerator

	// Cache for XDS resources
	Cache model.XdsCache
}

func NewDiscoveryServer(env *model.Environment) *DiscoveryServer {
	server := &DiscoveryServer{

		Generators:         map[string]XdsResourceGenerator{},
		DiscoveryStartTime: time.Now(),

		InboundUpdates:   atomic.NewInt64(0),
		CommittedUpdates: atomic.NewInt64(0),
	}

	return server
}

func (s *DiscoveryServer) Start(stopCh <-chan struct{}) {

	go s.sendPushes(stopCh)

}

// Register adds the ADS handler to the grpc server
func (s *DiscoveryServer) Register(rpcs *grpc.Server) {
	// Register v3 server
	discovery.RegisterAggregatedDiscoveryServiceServer(rpcs, s)
}

// CachesSynced is called when caches have been synced so that server can accept connections.
func (s *DiscoveryServer) CachesSynced() {
	klog.Infof("All caches have been synced up in %v, marking server ready", time.Since(s.DiscoveryStartTime))
	s.serverReady.Store(true)
}

func (s *DiscoveryServer) IsServerReady() bool {
	return s.serverReady.Load()
}

func (s *DiscoveryServer) WaitForRequestLimit(ctx context.Context) error {
	if s.RequestRateLimit.Limit() == 0 {
		// Allow opt out when rate limiting is set to 0qps
		return nil
	}
	// Give a bit of time for queue to clear out, but if not fail fast. Client will connect to another
	// instance in best case, or retry with backoff.
	wait, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return s.RequestRateLimit.Wait(wait)
}
