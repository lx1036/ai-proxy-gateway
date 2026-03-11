package xds

import "github.com/lx1036/gateway/pkg/model"

// WatchedResource tracks an active DiscoveryRequest subscription.
type WatchedResource struct {
	// TypeUrl is copied from the DiscoveryRequest.TypeUrl that initiated watching this resource.
	// nolint
	TypeUrl string
}

// XdsResourceGenerator creates the response for a typeURL DiscoveryRequest or DeltaDiscoveryRequest. If no generator
// is associated with a Proxy, the default (a networking.core.ConfigGenerator instance) will be used.
// The server may associate a different generator based on client metadata. Different
// WatchedResources may use same or different Generator.
// Note: any errors returned will completely close the XDS stream. Use with caution; typically and empty
// or no response is preferred.
type XdsResourceGenerator interface {
	// Generate generates the Sotw resources for Xds.
	Generate(proxy *model.Proxy, w *WatchedResource, req *model.PushRequest) (Resources, XdsLogDetails, error)
}
