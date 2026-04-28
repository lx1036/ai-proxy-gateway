package message

import (
	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	discoveryv1 "k8s.io/api/discovery/v1"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayAPIResources []*GatewayAPIResource

// DeepCopy INFO: 需要加上 DeepCopy 才能正常 watchable
func (c *GatewayAPIResources) DeepCopy() *GatewayAPIResources {
	if c == nil {
		return nil
	}
	out := make(GatewayAPIResources, len(*c))
	for i, res := range *c {
		if res != nil {
			out[i] = res.DeepCopy()
		}
	}

	return &out
}

type GatewayAPIResource struct {
	GatewayClass *gatewayapiv1.GatewayClass `json:"gatewayClass,omitempty" yaml:"gatewayClass,omitempty"`
	Gateways     []*gatewayapiv1.Gateway    `json:"gateways,omitempty" yaml:"gateways,omitempty"`
	HTTPRoutes   []*gatewayapiv1.HTTPRoute  `json:"httpRoutes,omitempty" yaml:"httpRoutes,omitempty"`

	EndpointSlices []*discoveryv1.EndpointSlice `json:"endpointSlices,omitempty" yaml:"endpointSlices,omitempty"`

	EnvoyExtensionPolicies []*envoygatewayv1alpha1.EnvoyExtensionPolicy `json:"envoyExtensionPolicies,omitempty" yaml:"envoyExtensionPolicies,omitempty"`

	EnvoyPatchPolicies []*envoygatewayv1alpha1.EnvoyPatchPolicy `json:"envoyPatchPolicies,omitempty" yaml:"envoyPatchPolicies,omitempty"`
}

func NewGatewayAPIResource() *GatewayAPIResource {
	return &GatewayAPIResource{}
}

// DeepCopy INFO: 需要加上 DeepCopy 才能正常 watchable
func (in *GatewayAPIResource) DeepCopy() *GatewayAPIResource {
	if in == nil {
		return nil
	}
	out := new(GatewayAPIResource)
	in.DeepCopyInto(out)
	return out
}

func (in *GatewayAPIResource) DeepCopyInto(out *GatewayAPIResource) {
	*out = *in

	if in.GatewayClass != nil {
		in, out := &in.GatewayClass, &out.GatewayClass
		*out = new(gatewayapiv1.GatewayClass)
		(*in).DeepCopyInto(*out)
	}

	if in.Gateways != nil {
		in, out := &in.Gateways, &out.Gateways
		*out = make([]*gatewayapiv1.Gateway, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(gatewayapiv1.Gateway)
				(*in).DeepCopyInto(*out)
			}
		}
	}

	if in.HTTPRoutes != nil {
		in, out := &in.HTTPRoutes, &out.HTTPRoutes
		*out = make([]*gatewayapiv1.HTTPRoute, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(gatewayapiv1.HTTPRoute)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}
