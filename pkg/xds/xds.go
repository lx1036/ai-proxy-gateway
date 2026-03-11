package xds

import (
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

type Resources = []*discovery.Resource

func ResourcesToAny(r Resources) []*anypb.Any {
	a := make([]*anypb.Any, 0, len(r))
	for _, rr := range r {
		a = append(a, rr.Resource)
	}
	return a
}
