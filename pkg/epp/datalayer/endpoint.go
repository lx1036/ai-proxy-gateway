package datalayer

import "k8s.io/apimachinery/pkg/types"

// EndpointMetadata represents the relevant Kubernetes Pod state of an inference server.
type EndpointMetadata struct {
	NamespacedName types.NamespacedName
	PodName        string
	Address        string
	Port           string
	MetricsHost    string
	Labels         map[string]string
}

func (e *EndpointMetadata) GetMetricsHost() string {
	return e.MetricsHost
}
