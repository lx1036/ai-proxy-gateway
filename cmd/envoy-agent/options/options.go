package options

import (
	"github.com/lx1036/gateway/pkg/agent"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"os"
)

type ProxyArgs struct {
	PodName      string
	PodNamespace string

	MeshConfigFile string
	// proxy config flags (named identically)
	ServiceCluster         string
	ProxyLogLevel          string
	ProxyComponentLogLevel string
	Concurrency            int

	DNSDomain string
}

func NewProxyArgs() ProxyArgs {
	p := ProxyArgs{
		PodName:      os.Getenv("POD_NAME"),
		PodNamespace: os.Getenv("POD_NAMESPACE"),
	}

	return p
}

func NewAgentOptions(proxy *ProxyArgs, cfg *meshconfig.ProxyConfig) *agent.AgentConfig {

	return &agent.AgentConfig{}
}
