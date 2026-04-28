package ir

import (
	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
)

// Infra
// +k8s:deepcopy-gen=true
type Infra struct {
	Proxy *ProxyInfra
}

// ProxyInfra
// +k8s:deepcopy-gen=true
type ProxyInfra struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`

	Metadata *InfraMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	EnvoyProxy *envoygatewayv1alpha1.EnvoyProxy
}

func NewProxyInfra() *ProxyInfra {
	return &ProxyInfra{
		Name:     "default",
		Metadata: NewInfraMetadata(),
	}
}

func (proxyInfra *ProxyInfra) GetEnvoyProxy() *envoygatewayv1alpha1.EnvoyProxy {
	if proxyInfra.EnvoyProxy == nil {
		proxyInfra.EnvoyProxy = new(envoygatewayv1alpha1.EnvoyProxy)
	}

	return proxyInfra.EnvoyProxy
}

// InfraMetadata
// +k8s:deepcopy-gen=true
type InfraMetadata struct {
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`

	OwnerReferences *ResourceMetadata `json:"ownerReference,omitempty" yaml:"ownerReference,omitempty"`
}

func NewInfraMetadata() *InfraMetadata {
	return &InfraMetadata{
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
	}
}

// ResourceMetadata
// +k8s:deepcopy-gen=true
type ResourceMetadata struct {
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	Name string `json:"name" yaml:"name"`

	Namespace string `json:"namespace" yaml:"namespace"`
}

func NewInfra() *Infra {
	return &Infra{
		Proxy: NewProxyInfra(),
	}
}
