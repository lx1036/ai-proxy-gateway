package core

import (
	"github.com/lx1036/gateway/pkg/model"
)

// ConfigGenerator represents the interfaces to be implemented by code that generates xDS responses
//type ConfigGenerator interface {
//	// BuildListeners returns the list of inbound/outbound listeners for the given proxy. This is the LDS output
//	// Internally, the computation will be optimized to ensure that listeners are computed only
//	// once and shared across multiple invocations of this function.
//	BuildListeners(node *model.Proxy, push *model.PushContext) []*listener.Listener
//
//	// BuildClusters returns the list of clusters for the given proxy. This is the CDS output
//	BuildClusters(node *model.Proxy, req *model.PushRequest) ([]*discovery.Resource, model.XdsLogDetails)
//
//	// BuildDeltaClusters returns both a list of resources that need to be pushed for a given proxy and a list of resources
//	// that have been deleted and should be removed from a given proxy. This is Delta CDS output.
//	BuildDeltaClusters(proxy *model.Proxy, updates *model.PushRequest,
//		watched *model.WatchedResource) ([]*discovery.Resource, []string, model.XdsLogDetails, bool)
//
//	// BuildHTTPRoutes returns the list of HTTP routes for the given proxy. This is the RDS output
//	BuildHTTPRoutes(node *model.Proxy, req *model.PushRequest, routeNames []string) ([]*discovery.Resource, model.XdsLogDetails)
//
//	// BuildNameTable returns list of hostnames and the associated IPs
//	BuildNameTable(node *model.Proxy, push *model.PushContext) *dnsProto.NameTable
//
//	// BuildExtensionConfiguration returns the list of extension configuration for the given proxy and list of names. This is the ECDS output.
//	BuildExtensionConfiguration(node *model.Proxy, push *model.PushContext, extensionConfigNames []string,
//		pullSecrets map[string][]byte) []*core.TypedExtensionConfig
//
//	// MeshConfigChanged is invoked when mesh config is changed, giving a chance to rebuild any cached config.
//	MeshConfigChanged(mesh *meshconfig.MeshConfig)
//}

type ConfigGenerator struct {
	//Cache model.XdsCache

}

func NewConfigGenerator(cache model.XdsCache) *ConfigGenerator {
	return &ConfigGenerator{
		//Cache: cache,
	}
}
