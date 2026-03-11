package translation

import (
	"istio.io/istio/pkg/kube/krt"
	"sync"

	ingressconfig "github.com/lx1036/gateway/pkg/core/ingress/config"
	"github.com/lx1036/gateway/pkg/core/ingress/kube/common"
	"github.com/lx1036/gateway/pkg/core/kube"

	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/collection"
)

type IngressTranslation struct {
	mutex sync.RWMutex

	ingressConfig *ingressconfig.IngressConfig

	//higressRouteCache  model.IngressRouteCollection
	//higressDomainCache model.IngressDomainCollection
}

func NewIngressTranslation(localKubeClient kube.Client, xdsUpdater model.XDSUpdater, namespace string, options common.Options) *IngressTranslation {
	if options.ClusterId == "Kubernetes" {
		options.ClusterId = ""
	}

	Config := &IngressTranslation{
		ingressConfig: ingressconfig.NewIngressConfig(localKubeClient, xdsUpdater, namespace, options),
	}

	return Config
}

func (m *IngressTranslation) Run(stop <-chan struct{}) {
	go m.ingressConfig.Run(stop)

}

func (m *IngressTranslation) AddLocalCluster(options common.Options) {
	m.ingressConfig.AddLocalCluster(options)
}

func (m *IngressTranslation) RegisterEventHandler(kind config.GroupVersionKind, handler model.EventHandler) {
	m.ingressConfig.RegisterEventHandler(kind, handler)
}

func (m *IngressTranslation) HasSynced() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if !m.ingressConfig.HasSynced() {
		return false
	}

	return true
}

func (m *IngressTranslation) Schemas() collection.Schemas {
	return common.IngressIR // TODO: 作用是什么
}

func (m *IngressTranslation) Get(typ config.GroupVersionKind, name, namespace string) *config.Config {
	//TODO implement me
	panic("implement me")
}

func (m *IngressTranslation) List(typ config.GroupVersionKind, namespace string) []config.Config {
	//TODO implement me
	panic("implement me")
}

func (m *IngressTranslation) Create(config config.Config) (revision string, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *IngressTranslation) Update(config config.Config) (newRevision string, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *IngressTranslation) UpdateStatus(config config.Config) (newRevision string, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *IngressTranslation) Delete(typ config.GroupVersionKind, name, namespace string, resourceVersion *string) error {
	//TODO implement me
	panic("implement me")
}

func (m *IngressTranslation) KrtCollection(kind config.GroupVersionKind) krt.Collection[config.Config] {
	//TODO implement me
	return nil
}
