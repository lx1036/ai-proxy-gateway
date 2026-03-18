package extensionserver

import (
	"github.com/envoyproxy/gateway/proto/extension"
)

/**
@see https://gateway.envoyproxy.io/docs/tasks/extensibility/extension-server/
*/

// Server is the implementation of the EnvoyGatewayExtensionServer interface.
type Server struct {
	extension.UnimplementedEnvoyGatewayExtensionServer
}
