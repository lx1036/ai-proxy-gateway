package common

import (
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/collections"
)

var IngressIR = collection.NewSchemasBuilder().
	MustAdd(collections.DestinationRule).
	MustAdd(collections.EnvoyFilter).
	MustAdd(collections.Gateway).
	MustAdd(collections.ServiceEntry).
	MustAdd(collections.VirtualService).
	MustAdd(collections.WasmPlugin).
	Build()
