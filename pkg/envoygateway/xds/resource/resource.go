package resource

import (
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

)

type XdsResources = map[resourcev3.Type][]types.Resource

type ResourceVersionTable struct {
	XdsResources
	// EnvoyPatchPolicyStatuses
}
