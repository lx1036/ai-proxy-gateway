package kube

import (
	"istio.io/istio/pkg/cluster"
	istiokube "istio.io/istio/pkg/kube"
	"k8s.io/client-go/tools/clientcmd"
)

type Client interface {
	istiokube.Client

	// Higress returns the Higress kube client.
	//Higress() higressclient.Interface

	// HigressInformer returns an informer for the higress client
	//HigressInformer() higressinformer.SharedInformerFactory

	// KIngress return the Knative kube client
	//KIngress() kingressclient.Interface

	//KIngressInformer() kingressinformer.SharedInformerFactory
}

type client struct {
	istiokube.Client
}

func NewClient(clientConfig clientcmd.ClientConfig, cluster cluster.ID) (Client, error) {
	var c client
	istioClient, err := istiokube.NewClient(clientConfig, cluster)
	if err != nil {
		return nil, err
	}
	c.Client = istioClient

	return &c, nil
}

// EnableCrdWatcher enables the CRD watcher on the client.
func EnableCrdWatcher(c Client) Client {
	istiokube.EnableCrdWatcher(c.(*client).Client)
	return c
}
