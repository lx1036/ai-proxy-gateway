package grpc

import (
	gatewayKeepalive "github.com/lx1036/gateway/pkg/keepalive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func ServerOptions(options *gatewayKeepalive.Options, interceptors ...grpc.UnaryServerInterceptor) []grpc.ServerOption {
	maxStreams := 100000
	maxRecvMsgSize := 4 * 1024 * 1024

	grpcOptions := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(interceptors...),
		grpc.MaxConcurrentStreams(uint32(maxStreams)),
		grpc.MaxRecvMsgSize(maxRecvMsgSize),
		// Ensure we allow clients sufficient ability to send keep alives. If this is higher than client
		// keep alive setting, it will prematurely get a GOAWAY sent.
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime: options.Time / 2,
		}),
		//grpc.StatsHandler(statsHandler(handleNewMaxMessageSize)),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:                  options.Time,
			Timeout:               options.Timeout,
			MaxConnectionAge:      options.MaxServerConnectionAge,
			MaxConnectionAgeGrace: options.MaxServerConnectionAgeGrace,
		}),
	}

	return grpcOptions
}
