package kube

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/klog/v2"
)

// clientFactory partially implements the kubectl util.Factory, which is provides access to various k8s clients.
// The full Factory can be built with MakeKubeFactory.
// This split is to avoid huge dependencies.
type clientFactory struct {
	clientConfig clientcmd.ClientConfig

	expander meta.RESTMapper
	mapper   meta.ResettableRESTMapper

	discoveryClient discovery.CachedDiscoveryInterface
}

func newClientFactory(clientConfig clientcmd.ClientConfig) *clientFactory {
	factory := &clientFactory{
		clientConfig: clientConfig,
	}

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		klog.Fatalf("Failed to create REST config: %v", err)
	}

	d, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		klog.Fatalf("Failed to create discovery client: %v", err)
	}
	factory.discoveryClient = memory.NewMemCacheClient(d)

	factory.mapper = restmapper.NewDeferredDiscoveryRESTMapper(factory.discoveryClient)
	factory.expander = restmapper.NewShortcutExpander(factory.mapper, factory.discoveryClient, func(string) {})

	return factory
}

func (c *clientFactory) ToRESTConfig() (*rest.Config, error) {
	restConfig, err := c.clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return SetRestDefaults(restConfig), nil
}
