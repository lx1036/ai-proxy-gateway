package istio

import (
	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/serviceregistry/kube/controller"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/config/schema/gvr"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/kclient"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/kube/kubetypes"
	corev1 "k8s.io/api/core/v1"
	inferencev1 "sigs.k8s.io/gateway-api-inference-extension/api/v1"
	gateway "sigs.k8s.io/gateway-api/apis/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

/**
会把 gateway-api 翻译成 (Istio schema Gateway/VirtualService)
*/

type Inputs struct {
	Namespaces krt.Collection[*corev1.Namespace]

	GatewayClasses krt.Collection[*gateway.GatewayClass]
	Gateways       krt.Collection[*gateway.Gateway]
	HTTPRoutes     krt.Collection[*gateway.HTTPRoute]
}

// Controller defines the controller for the gateway-api. The controller reads a variety of resources (Gateway types, as well
// as adjacent types like Namespace and Service), and through `krt`, translates them into Istio types (Gateway/VirtualService).

// The status on all gateway-api types is also tracked. Each collection emits downstream objects, but also status about the
// input type. If the status changes, it is queued to asynchronously update the status of the object in Kubernetes.

type Controller struct {
	DefaultGatewaySelector map[string]string

	// revision the controller is running under
	revision string
}

func NewController(
	kc kube.Client,
	waitForCRD func(class schema.GroupVersionResource, stop <-chan struct{}) bool,
	options controller.Options,
	xdsUpdater model.XDSUpdater,
) *Controller {

	c := &Controller{
		revision: options.Revision,
	}

	inputs := Inputs{
		Namespaces: krt.NewInformer[*corev1.Namespace](kc, opts.WithName("informer/Namespaces")...),

		GatewayClasses: buildClient[*gateway.GatewayClass](c, kc, gvr.GatewayClass, opts, "informer/GatewayClasses"),
		Gateways:       buildClient[*gateway.Gateway](c, kc, gvr.KubernetesGateway, opts, "informer/Gateways"),
		HTTPRoutes:     buildClient[*gateway.HTTPRoute](c, kc, gvr.HTTPRoute, opts, "informer/HTTPRoutes"),
	}

	if features.EnableGatewayAPIInferenceExtension {
		inputs.InferencePools = buildClient[*inferencev1.InferencePool](c, kc, gvr.InferencePool, opts, "informer/InferencePools")
	} else {
		// If disabled, still build a collection but make it always empty
		inputs.InferencePools = krt.NewStaticCollection[*inferencev1.InferencePool](nil, nil, opts.WithName("disable/InferencePools")...)
	}

	return c
}

func (c *Controller) Run(stop <-chan struct{}) {
	if features.EnableGatewayAPIGatewayClassController {
		go func() {
			if c.waitForCRD(gvr.GatewayClass, stop) {
				gcc := NewClassController(c.client)
				c.client.RunAndWait(stop) // ???
				gcc.Run(stop)
			}
		}()
	}

	<-stop
	close(c.stop)
}

// buildClient is a small wrapper to build a krt collection based on a delayed informer.
func buildClient[I controllers.ComparableObject](
	c *Controller,
	kc kube.Client,
	res schema.GroupVersionResource,
	opts krt.OptionsBuilder,
	name string,
) krt.Collection[I] {
	filter := kclient.Filter{
		ObjectFilter: kubetypes.ComposeFilters(kc.ObjectFilter(), c.inRevision),
	}

	// all other types are filtered by revision, but for gateways we need to select tags as well
	if res == gvr.KubernetesGateway {
		filter.ObjectFilter = kc.ObjectFilter()
	}

	cc := kclient.NewDelayedInformer[I](kc, res, kubetypes.StandardInformer, filter)
	return krt.WrapClient[I](cc, opts.WithName(name)...)
}

func (c *Controller) inRevision(obj any) bool {
	object := controllers.ExtractObject(obj)
	if object == nil {
		return false
	}
	return config.LabelsInRevision(object.GetLabels(), c.revision)
}
