package kube

import (
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	kubeExtClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubeVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	gatewayapiinferenceclient "sigs.k8s.io/gateway-api-inference-extension/client-go/clientset/versioned"
	gatewayapiclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

// Client is a helper for common Kubernetes client operations. This contains various different kubernetes
// clients using a shared config. It is expected that all of Istiod can share the same set of clients and
// informers. Sharing informers is especially important for load on the API server/Istiod itself.
type Client interface {
	// RESTConfig returns the Kubernetes rest.Config used to configure the clients.
	RESTConfig() *rest.Config

	// Ext returns the API extensions client.
	Ext() kubeExtClient.Interface

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

	// GatewayAPIInference returns the gateway-api kube client.
	GatewayAPIInference() gatewayapiinferenceclient.Interface

	// Informers returns an informer factory
	Informers() informerfactory.InformerFactory

	// CrdWatcher returns the CRD watcher for this client
	CrdWatcher() kubetypes.CrdWatcher

	// ObjectFilter returns an object filter that can be used to filter out unwanted objects based on configuration.
	// This must be set on a client with SetObjectFilter.
	ObjectFilter() kubetypes.DynamicObjectFilter

	// RunAndWait starts all informers and waits for their caches to sync.
	// Warning: this must be called AFTER .Informer() is called, which will register the informer.
	// "false" is returned if this prematurely exited without syncing.
	RunAndWait(stop <-chan struct{}) bool

	// WaitForCacheSync waits for all cache functions to sync, as well as all informers started by the *fake* client.
	WaitForCacheSync(name string, stop <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool

	// GetKubernetesVersion returns the Kubernetes server version
	GetKubernetesVersion() (*kubeVersion.Info, error)

	// Shutdown closes all informers and waits for them to terminate
	Shutdown()

	// ClusterID returns the cluster this client is connected to
	ClusterID() cluster.ID

	// IsWatchListSemanticsUnSupported is used by internal client-go libraries to tell if the client is a fake client (more or less)
	IsWatchListSemanticsUnSupported() bool
}
