package kubeclient

import (
	"github.com/lx1036/gateway/pkg/config/schema/kubetypes"
	"github.com/lx1036/gateway/pkg/kube/informerfactory"
	ktypes "github.com/lx1036/gateway/pkg/kube/kubetypes"
	"github.com/lx1036/gateway/pkg/typemap"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	kubeext "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/tools/cache"
	gatewayapiinferenceclient "sigs.k8s.io/gateway-api-inference-extension/client-go/clientset/versioned"
	gatewayapiclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

type ClientGetter interface {
	// Ext returns the API extensions client.
	Ext() kubeext.Interface

	// Kube returns the core kube client
	Kube() kubernetes.Interface

	// Dynamic client.
	Dynamic() dynamic.Interface

	// Metadata returns the Metadata kube client.
	Metadata() metadata.Interface

	// Istio returns the Istio kube client.
	Istio() istioclient.Interface

	// GatewayAPI returns the gateway-api kube client.
	GatewayAPI() gatewayapiclient.Interface

	// GatewayAPIInference returns the gateway-api-inference-extension kube client.
	GatewayAPIInference() gatewayapiinferenceclient.Interface

	// Informers returns an informer factory.
	Informers() informerfactory.InformerFactory

	// IsWatchListSemanticsUnSupported is used by internal client-go libraries to tell if the client is a fake client (more or less)
	IsWatchListSemanticsUnSupported() bool
}

var registerTypes = typemap.NewTypeMap()

// TypeRegistration represents the necessary methods
// to provide a custom type to the kubeclient informer mechanism
type TypeRegistration[T runtime.Object] interface {
	kubetypes.RegisterType[T]

	// ListWatch provides the necessary methods for list and
	// watch for the informer
	ListWatch(c ClientGetter, opts ktypes.InformerOptions) cache.ListerWatcher

	// Write returns a writer interface. This may return nil if the registration is not writeable
	Write(c ClientGetter, namespace string) ktypes.WriteAPI[T]
}

func setupInformer(opts ktypes.InformerOptions, inf cache.SharedIndexInformer) {
	// It is important to set this in the newFunc rather than after InformerFor to avoid
	// https://github.com/kubernetes/kubernetes/issues/117869
	if opts.ObjectTransform != nil {
		_ = inf.SetTransform(opts.ObjectTransform)
	} else {
		_ = inf.SetTransform(stripUnusedFields)
	}

	//if err := inf.SetWatchErrorHandler(informermetric.ErrorHandlerForCluster(opts.Cluster)); err != nil {
	//	klog.Errorf("failed to set watch handler, informer may already be started: %v", err)
	//}
}

// stripUnusedFields is the transform function for shared informers,
// it removes unused fields from objects before they are stored in the cache to save memory.
func stripUnusedFields(obj any) (any, error) {
	t, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		// shouldn't happen
		return obj, nil
	}
	// ManagedFields is large and we never use it
	t.GetObjectMeta().SetManagedFields(nil)
	return obj, nil
}
