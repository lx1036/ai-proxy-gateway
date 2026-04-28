package ir

// RouteDestination holds the destination details associated with the route
// +kubebuilder:object:generate=true
type RouteDestination struct {
	// Name of the destination. This field allows the xds layer
	// to check if this route destination already exists and can be
	// reused
	Name     string                `json:"name" yaml:"name"`
	StatName *string               `json:"statName,omitempty" yaml:"statName,omitempty"`
	Settings []*DestinationSetting `json:"settings,omitempty" yaml:"settings,omitempty"`
	// Metadata is used to enrich envoy route metadata with user and provider-specific information
	// RouteDestination metadata is primarily derived from the xRoute resources. In some cases,
	// the primary resource is a Policy or Envoy Proxy, when non-xRoute backendRefs are used.
	Metadata *ResourceMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// DestinationSetting holds the settings associated with the destination
// +kubebuilder:object:generate=true
type DestinationSetting struct {
	// Name of the setting
	Name string `json:"name" yaml:"name"`

	Endpoints []*DestinationEndpoint `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
}

// DestinationEndpoint holds the endpoint details associated with the destination
// +kubebuilder:object:generate=true
type DestinationEndpoint struct {
	// Hostname refers to the endpoint's hostname
	Hostname *string `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	// Host refers to the FQDN or IP address of the backend service.
	Host string `json:"host" yaml:"host"`
	// Port on the service to forward the request to.
	Port uint32 `json:"port" yaml:"port"`
	// Path refers to the Unix Domain Socket
	Path *string `json:"path,omitempty" yaml:"path,omitempty"`
	// Draining is true if this endpoint should be drained
	Draining bool `json:"draining,omitempty" yaml:"draining,omitempty"`
	// Zone refers to the topology zone the Endpoint resides in
	Zone *string `json:"zone,omitempty" yaml:"zone,omitempty"`
}
