package model

import (
	"sync"
)

type Environment struct {
	// Discovery interface for listing services and instances.
	//ServiceDiscovery

	// Config interface for listing routing rules
	ConfigStore

	// Watcher is the watcher for the mesh config (to be merged into the config store)
	//Watcher

	// NetworksWatcher (loaded from a config map) provides information about the
	// set of networks inside a mesh and how to route to endpoints in each
	// network. Each network provides information about the endpoints in a
	// routable L3 network. A single routable L3 network can have one or more
	// service registries.
	//NetworksWatcher mesh.NetworksWatcher
	//
	//NetworkManager *NetworkManager

	// mutex used for protecting Environment.pushContext
	mutex sync.RWMutex
	// pushContext holds information during push generation. It is reset on config change, at the beginning
	// of the pushAll. It will hold all errors and stats and possibly caches needed during the entire cache computation.
	// DO NOT USE EXCEPT FOR TESTS AND HANDLING OF NEW CONNECTIONS.
	// ALL USE DURING A PUSH SHOULD USE THE ONE CREATED AT THE
	// START OF THE PUSH, THE GLOBAL ONE MAY CHANGE AND REFLECT A DIFFERENT
	// CONFIG AND PUSH
	//pushContext *PushContext

	// DomainSuffix provides a default domain for the Istio server.
	DomainSuffix string

	// TrustBundle: List of Mesh TrustAnchors
	//TrustBundle *trustbundle.TrustBundle
	//
	//clusterLocalServices ClusterLocalProvider
	//
	//CredentialsController credentials.MulticlusterController
	//
	//GatewayAPIController GatewayController
	//
	//// EndpointShards for a service. This is a global (per-server) list, built from
	//// incremental updates. This is keyed by service and namespace
	//EndpointIndex *EndpointIndex
	//
	//// Cache for XDS resources.
	//Cache XdsCache
}

func NewEnvironment() *Environment {
	//cache := NewXdsCache()
	return &Environment{
		//pushContext:   NewPushContext(),
		//Cache:         cache,
		//EndpointIndex: NewEndpointIndex(cache),
	}
}

// Proxy contains information about an specific instance of a proxy (envoy sidecar, gateway,
// etc). The Proxy is initialized when a sidecar connects to Pilot, and populated from
// 'node' info in the protocol as well as data extracted from registries.
//
// In current Istio implementation nodes use a 4-parts '~' delimited ID.
// Type~IPAddress~ID~Domain
type Proxy struct {
	sync.RWMutex
}
