package kubernetes

import "k8s.io/apimachinery/pkg/util/sets"

type resourceMappings struct {
	// Set for storing namespaces for Route, Service and Gateway objects.
	allAssociatedNamespaces sets.Set[string]

	// Set for storing Gateways' NamespacedNames.
	allAssociatedGateways sets.Set[string]

	// Set for storing HTTPRoutes' NamespacedNames attaching to various Gateway objects.
	allAssociatedHTTPRoutes sets.Set[string]
}


func newResourceMapping() *resourceMappings {
	return &resourceMappings{
		allAssociatedNamespaces:                 sets.New[string](),
		allAssociatedGateways:                   sets.New[string](),
		allAssociatedHTTPRoutes:                 sets.New[string](),


	}

}
