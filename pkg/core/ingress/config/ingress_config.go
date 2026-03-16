package config

import (
	"github.com/lx1036/gateway/pkg/core/ingress/kube/common"
	"github.com/lx1036/gateway/pkg/core/ingress/kube/gateway"
	"github.com/lx1036/gateway/pkg/core/kube"
	"k8s.io/klog/v2"
	"sync"

	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/config"
)

type IngressConfig struct {
	mutex sync.RWMutex

	localKubeClient          kube.Client
	remoteGatewayControllers map[cluster.ID]common.GatewayController

	XDSUpdater istiomodel.XDSUpdater
}

// NewIngressConfig creates a new ConfigStore
func NewIngressConfig(localKubeClient kube.Client, xdsUpdater istiomodel.XDSUpdater, namespace string, options common.Options) *IngressConfig {
	clusterId := options.ClusterId
	if clusterId == "Kubernetes" {
		clusterId = ""
	}

	cfg := &IngressConfig{
		localKubeClient: localKubeClient,
		XDSUpdater:      xdsUpdater,

		remoteGatewayControllers: make(map[cluster.ID]common.GatewayController),
	}

	return cfg
}

func (m *IngressConfig) Run(stop <-chan struct{}) {

	for _, remoteGatewayController := range m.remoteGatewayControllers {
		//_ = remoteGatewayController.SetWatchErrorHandler(m.watchErrorHandler)
		go remoteGatewayController.Run(stop)
	}

}

func (m *IngressConfig) AddLocalCluster(options common.Options) {
	m.remoteGatewayControllers[options.ClusterId] = gateway.NewController(m.localKubeClient, options, m.XDSUpdater)
}

func (m *IngressConfig) RegisterEventHandler(kind config.GroupVersionKind, handler istiomodel.EventHandler) {
	klog.Infof("register resource %s", kind.String())

	for _, remoteGatewayController := range m.remoteGatewayControllers {
		remoteGatewayController.RegisterEventHandler(kind, handler)
	}
}

func (m *IngressConfig) HasSynced() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, remoteGatewayController := range m.remoteGatewayControllers {
		if !remoteGatewayController.HasSynced() {
			return false
		}
	}

	klog.Infof("Ingress config controller synced.")
	return true
}
