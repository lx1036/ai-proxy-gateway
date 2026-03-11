package common

import "istio.io/istio/pkg/cluster"

type Options struct {
	Enable               bool
	ClusterId            cluster.ID
	IngressClass         string
	GatewayClass         string
	WatchNamespace       string
	RawClusterId         string
	EnableStatus         bool
	SystemNamespace      string
	GatewaySelectorKey   string
	GatewaySelectorValue string
	GatewayHttpPort      uint32
	GatewayHttpsPort     uint32
}
