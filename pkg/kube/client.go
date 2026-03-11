package kube

import (
	"net/http"

	"github.com/lx1036/gateway/pkg/kube/informerfactory"
	"github.com/lx1036/gateway/pkg/kube/kubetypes"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	kubeExtClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	kubeVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	gatewayapiinferenceclient "sigs.k8s.io/gateway-api-inference-extension/client-go/clientset/versioned"
	gatewayapiclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	"time"
)

// Client is a helper for common Kubernetes client operations. This contains various different kubernetes
// clients using a shared config. It is expected that all of Istiod can share the same set of clients and
// informers. Sharing informers is especially important for load on the API server/Istiod itself.
type Client interface {
	// RESTConfig returns the Kubernetes rest.Config used to configure the clients.
	//RESTConfig() *rest.Config
	//
	//// Ext returns the API extensions client.
	//Ext() kubeExtClient.Interface
	//
	//// Kube returns the core kube client
	//Kube() kubernetes.Interface
	//
	//// Dynamic client.
	//Dynamic() dynamic.Interface
	//
	//// Metadata returns the Metadata kube client.
	//Metadata() metadata.Interface
	//
	//// Istio returns the Istio kube client.
	//Istio() istioclient.Interface
	//
	//// GatewayAPI returns the gateway-api kube client.
	//GatewayAPI() gatewayapiclient.Interface
	//
	//// GatewayAPIInference returns the gateway-api kube client.
	//GatewayAPIInference() gatewayapiinferenceclient.Interface
	//
	//// Informers returns an informer factory
	//Informers() informerfactory.InformerFactory
	//
	//// CrdWatcher returns the CRD watcher for this client
	//CrdWatcher() kubetypes.CrdWatcher
	//
	//// ObjectFilter returns an object filter that can be used to filter out unwanted objects based on configuration.
	//// This must be set on a client with SetObjectFilter.
	////ObjectFilter() kubetypes.DynamicObjectFilter
	//
	//// RunAndWait starts all informers and waits for their caches to sync.
	//// Warning: this must be called AFTER .Informer() is called, which will register the informer.
	//// "false" is returned if this prematurely exited without syncing.
	//RunAndWait(stop <-chan struct{}) bool
	//
	//// WaitForCacheSync waits for all cache functions to sync, as well as all informers started by the *fake* client.
	//WaitForCacheSync(name string, stop <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool
	//
	//// GetKubernetesVersion returns the Kubernetes server version
	//GetKubernetesVersion() (*kubeVersion.Info, error)
	//
	//// Shutdown closes all informers and waits for them to terminate
	//Shutdown()

	// ClusterID returns the cluster this client is connected to
	//ClusterID() cluster.ID

	// IsWatchListSemanticsUnSupported is used by internal client-go libraries to tell if the client is a fake client (more or less)
	//IsWatchListSemanticsUnSupported() bool
}

type client struct {
	clientFactory *clientFactory
	config        *rest.Config

	informerFactory     informerfactory.InformerFactory
	extSet              kubeExtClient.Interface
	kube                kubernetes.Interface
	dynamic             dynamic.Interface
	metadata            metadata.Interface
	istio               istioclient.Interface
	gatewayapi          gatewayapiclient.Interface
	gatewayapiinference gatewayapiinferenceclient.Interface
	restClient          *rest.RESTClient
	discoveryClient     discovery.CachedDiscoveryInterface
	mapper              meta.ResettableRESTMapper

	version *kubeVersion.Info

	// http is a client for HTTP requests
	http *http.Client

	crdWatcher kubetypes.CrdWatcher
}

// NewClient creates a Kubernetes client from the given rest config.
func NewClient(clientConfig clientcmd.ClientConfig, cluster string) (Client, error) {
	return newClientInternal(newClientFactory(clientConfig))
}

// newClientInternal creates a Kubernetes client from the given factory.
func newClientInternal(clientFactory *clientFactory) (*client, error) {
	var c client
	var err error

	c.clientFactory = clientFactory

	c.config, err = clientFactory.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	//for _, opt := range opts {
	//	opt(&c)
	//}

	//c.restClient, err = clientFactory.RESTClient()
	//if err != nil {
	//	return nil, err
	//}
	//
	//c.discoveryClient, err = clientFactory.ToDiscoveryClient()
	//if err != nil {
	//	return nil, err
	//}
	//c.mapper, err = clientFactory.mapper.Get()
	//if err != nil {
	//	return nil, err
	//}
	//
	//c.informerFactory = informerfactory.NewSharedInformerFactory()

	// k8s client
	c.kube, err = kubernetes.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	c.metadata, err = metadata.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	// dynamic client
	c.dynamic, err = dynamic.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	// istio client
	c.istio, err = istioclient.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	// gateway-api client
	c.gatewayapi, err = gatewayapiclient.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	// gateway-api-inference-extension client
	c.gatewayapiinference, err = gatewayapiinferenceclient.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	// k8s CRD client
	c.extSet, err = kubeExtClient.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	c.http = &http.Client{}
	if c.config != nil && c.config.Timeout != 0 {
		c.http.Timeout = c.config.Timeout
	} else {
		c.http.Timeout = time.Second * 15
	}

	//c.version, err = c.kube.Discovery().ServerVersion()
	return &c, nil
}

var NewCrdWatcher func(Client) kubetypes.CrdWatcher

// EnableCrdWatcher enables the CRD watcher on the client.
func EnableCrdWatcher(c Client) Client {
	if NewCrdWatcher == nil {
		panic("NewCrdWatcher is unset. Likely the crd watcher library is not imported anywhere")
	}
	if c.(*client).crdWatcher != nil {
		panic("EnableCrdWatcher called twice for the same client")
	}
	c.(*client).crdWatcher = NewCrdWatcher(c)
	return c
}
