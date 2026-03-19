package extensionserver

import (
	"context"

	"github.com/envoyproxy/gateway/proto/extension"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

/**
@see https://gateway.envoyproxy.io/docs/tasks/extensibility/extension-server/

客户端代码在：@see /Users/lx1036/Code/k8s/envoyproxy-gateway/internal/extension/registry/xds_hook.go
*/

const serverName = "envoy-gateway-extension-server"

// Server is the implementation of the EnvoyGatewayExtensionServer interface.
type Server struct {
	extension.UnimplementedEnvoyGatewayExtensionServer
}

func New() *Server {
	return &Server{}
}

// Check implements [grpc_health_v1.HealthServer].
func (s *Server) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

// Watch implements [grpc_health_v1.HealthServer].
func (s *Server) Watch(*grpc_health_v1.HealthCheckRequest, grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "Watch is not implemented")
}

// List implements [grpc_health_v1.HealthServer].
func (s *Server) List(context.Context, *grpc_health_v1.HealthListRequest) (*grpc_health_v1.HealthListResponse, error) {
	return &grpc_health_v1.HealthListResponse{Statuses: map[string]*grpc_health_v1.HealthCheckResponse{
		serverName: {Status: grpc_health_v1.HealthCheckResponse_SERVING},
	}}, nil
}
