package agent

import (
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"k8s.io/klog/v2"
)

type XdsProxy struct {
	// AggregatedDiscoveryServiceServer 实现 xDS 协议，允许一个 gRPC 流处理所有类型的 xDS 请求（CDS、LDS、EDS、RDS 等），而不是为每种资源类型单独建立流
	discovery.UnimplementedAggregatedDiscoveryServiceServer
}

func (p *XdsProxy) StreamAggregatedResources(downstream discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	klog.Infof("accepted XDS connection from Envoy, forwarding to upstream XDS server")

	return nil
}
