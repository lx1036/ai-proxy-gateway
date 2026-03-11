package v3

const (
	APITypePrefix              = "type.googleapis.com/"
	ClusterType                = APITypePrefix + "envoy.config.cluster.v3.Cluster"
	EndpointType               = APITypePrefix + "envoy.config.endpoint.v3.ClusterLoadAssignment"
	ListenerType               = APITypePrefix + "envoy.config.listener.v3.Listener"
	RouteType                  = APITypePrefix + "envoy.config.route.v3.RouteConfiguration"
	SecretType                 = APITypePrefix + "envoy.extensions.transport_sockets.tls.v3.Secret"
	ExtensionConfigurationType = APITypePrefix + "envoy.config.core.v3.TypedExtensionConfig"
)
