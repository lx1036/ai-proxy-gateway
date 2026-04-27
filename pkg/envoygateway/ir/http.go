package ir



// HTTPListener
// +k8s:deepcopy-gen=true
type HTTPListener struct {
	Hostnames []string `json:"hostnames" yaml:"hostnames"`

	// Headers configures special header management for the listener
	//Headers *HeaderSettings `json:"headers,omitempty" yaml:"headers,omitempty"`

	// Routes associated with HTTP traffic to the service.
	Routes []*HTTPRoute `json:"routes,omitempty" yaml:"routes,omitempty"`
}

// HTTPRoute holds the route information associated with the HTTP Route
// +k8s:deepcopy-gen=true
type HTTPRoute struct {
	Name string `json:"name" yaml:"name"`

	// Hostname that the route matches against
	Hostname string `json:"hostname" yaml:"hostname,omitempty"`


	// EnvoyExtension holds the features associated with EnvoyExtensionPolicy
	EnvoyExtensions *EnvoyExtensionFeatures `json:"envoyExtensions,omitempty" yaml:"envoyExtensions,omitempty"`
}


