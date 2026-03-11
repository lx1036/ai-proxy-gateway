package gateway

import (
	"github.com/lx1036/gateway/pkg/core/ingress/kube/common"
	"github.com/lx1036/gateway/pkg/core/util"
	kubecontroller "istio.io/istio/pilot/pkg/serviceregistry/kube/controller"
	"istio.io/istio/pilot/pkg/status"
	"k8s.io/klog/v2"

	//istiogateway "github.com/lx1036/gateway/pkg/core/ingress/kube/gateway/istio"
	"github.com/lx1036/gateway/pkg/core/kube"
	"istio.io/istio/pkg/kube/krt"

	"istio.io/istio/pilot/pkg/config/kube/crdclient"
	istiogateway "istio.io/istio/pilot/pkg/config/kube/gateway"
	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/istio/pkg/config/schema/resource"
)

type gatewayController struct {
	store model.ConfigStoreController

	istioController *istiogateway.Controller
	statusManager   *status.Manager
}

func NewController(client kube.Client, options common.Options, xdsUpdater model.XDSUpdater) common.GatewayController {
	// add schema GatewayClass
	schemasBuilder := collection.NewSchemasBuilder()
	collections.PilotGatewayAPI().ForEach(func(schema resource.Schema) bool {
		if schema.Group() == collections.GatewayClass.Group() {
			schemasBuilder.MustAdd(schema)
		}
		return false
	})

	// add gateway-api-inference-extension, schema InferencePool
	if features.EnableGatewayAPIInferenceExtension {
		schemasBuilder.MustAdd(collections.InferencePool)
	}
	opts := crdclient.Option{
		Revision:     util.Revision,
		DomainSuffix: "cluster.local",
		Identifier:   "gateway-controller",
	}
	store := crdclient.NewForSchemas(client, opts, schemasBuilder.Build())

	opt := kubecontroller.Options{
		DomainSuffix: "cluster.local",
		ClusterID:    options.ClusterId,
		Revision:     util.Revision,
	}
	istioController := istiogateway.NewController(client, client.CrdWatcher().WaitForCRD, opt, xdsUpdater)
	if options.GatewaySelectorKey != "" {
		//istioController.DefaultGatewaySelector = map[string]string{options.GatewaySelectorKey: options.GatewaySelectorValue}
	}

	var statusManager *status.Manager = nil
	if options.EnableStatus {
		statusManager = status.NewManager(store)
		istioController.SetStatusWrite(true, statusManager)
	} else {
		klog.Infof("Disable status update for cluster %s", options.ClusterId)
	}

	return &gatewayController{
		store:           store,
		istioController: istioController,
		statusManager:   statusManager,
	}
}

func (g gatewayController) Run(stop <-chan struct{}) {
	g.store.Schemas().ForEach(func(schema resource.Schema) bool {
		g.store.RegisterEventHandler(schema.GroupVersionKind(), g.onEvent)
		return false
	})

	go g.store.Run(stop)
	go g.istioController.Run(stop)

	if g.statusManager != nil {
		g.statusManager.Start(stop)
	}
}

func (g *gatewayController) onEvent(prev config.Config, curr config.Config, event model.Event) {

}

func (g gatewayController) RegisterEventHandler(kind config.GroupVersionKind, handler model.EventHandler) {

}

func (g gatewayController) HasSynced() bool {
	ret := g.istioController.HasSynced()
	if ret {
		g.istioController.Reconcile(model.NewPushContext())
	}
	return ret
}

func (g gatewayController) Schemas() collection.Schemas {
	//TODO implement me
	panic("implement me")
}

func (g gatewayController) Get(typ config.GroupVersionKind, name, namespace string) *config.Config {
	//TODO implement me
	panic("implement me")
}

func (g gatewayController) List(typ config.GroupVersionKind, namespace string) []config.Config {
	//TODO implement me
	panic("implement me")
}

func (g gatewayController) Create(config config.Config) (revision string, err error) {
	//TODO implement me
	panic("implement me")
}

func (g gatewayController) Update(config config.Config) (newRevision string, err error) {
	//TODO implement me
	panic("implement me")
}

func (g gatewayController) UpdateStatus(config config.Config) (newRevision string, err error) {
	//TODO implement me
	panic("implement me")
}

func (g gatewayController) Delete(typ config.GroupVersionKind, name, namespace string, resourceVersion *string) error {
	//TODO implement me
	panic("implement me")
}

func (g gatewayController) KrtCollection(kind config.GroupVersionKind) krt.Collection[config.Config] {
	//TODO implement me
	panic("implement me")
}
