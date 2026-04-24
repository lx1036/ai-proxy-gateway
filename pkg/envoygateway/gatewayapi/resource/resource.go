package resource

import (
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type ControllerResources []*Resources

// DeepCopy INFO: 需要加上 DeepCopy 才能正常 watchable
func (c *ControllerResources) DeepCopy() *ControllerResources {
	if c == nil {
		return nil
	}
	out := make(ControllerResources, len(*c))
	for i, res := range *c {
		if res != nil {
			out[i] = res.DeepCopy()
		}
	}

	return &out
}

type Resources struct {
	GatewayClass *gatewayapiv1.GatewayClass `json:"gatewayClass,omitempty" yaml:"gatewayClass,omitempty"`
	Gateways     []*gatewayapiv1.Gateway    `json:"gateways,omitempty" yaml:"gateways,omitempty"`
	HTTPRoutes   []*gatewayapiv1.HTTPRoute  `json:"httpRoutes,omitempty" yaml:"httpRoutes,omitempty"`
}

func NewResources() *Resources {
	return &Resources{}
}

// DeepCopy INFO: 需要加上 DeepCopy 才能正常 watchable
func (in *Resources) DeepCopy() *Resources {
	if in == nil {
		return nil
	}
	out := new(Resources)
	in.DeepCopyInto(out)
	return out
}

func (in *Resources) DeepCopyInto(out *Resources) {
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
