package translator

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/xds/resource"
)

type Translator struct {
}

func (translator *Translator) Translate(xdsIR *ir.Xds) (*resource.ResourceVersionTable, error) {

	resourceTable := new(resource.ResourceVersionTable)

	translator.processHTTPListenerXdsTranslation(xdsIR, resourceTable)

	// Patch global resources that are shared across listeners and routes.
	// - the envoy client certificate
	// - the OIDC HMAC secret
	// - the rate limit server cluster
	if err := translator.patchGlobalResources(tCtx, xdsIR); err != nil {

	}

	// All XDS resources is ready, let's do the patch.
	if err := processJSONPatches(tCtx, xdsIR.EnvoyPatchPolicies); err != nil {

	}

	// Check if an extension want to inject any clusters/secrets
	// If no extension exists (or it doesn't subscribe to this hook) then this is a quick no-op
	if err := processExtensionPostTranslationHook(tCtx, t.ExtensionManager, xdsIR.ExtensionServerPolicies); err != nil {

	}

	return resourceTable, nil
}

func (translator *Translator) processHTTPListenerXdsTranslation(xdsIR *ir.Xds, resourceTable *resource.ResourceVersionTable) {

	for _, httpListener := range xdsIR.HTTP {

		// Search for an existing TCP listener on the same address + port combination.
		// Right now, the address is always 0.0.0.0/::, and we need to revisit the logic in the method if we want to support
		// listeners on specific addresses.
		tcpXDSListener = findXdsListenerByHostPort(tCtx, httpListener.Address, httpListener.Port, corev3.SocketAddress_TCP)
		quicXDSListener = findXdsListenerByHostPort(tCtx, httpListener.Address, httpListener.Port, corev3.SocketAddress_UDP)

		xdsListenerOnSameAddressPortExists = tcpXDSListener != nil
		tlsEnabled = httpListener.TLS != nil

		switch {
		case xdsListenerOnSameAddressPortExists && !tlsEnabled:
			addHCM = !hasHCMInDefaultFilterChain(tcpXDSListener)

		}

		if addHCM {
			translator.addHTTPConnectionManagerToXdsListener()
		} else {

		}

		resourceTable.AddXdsResource(resourcev3.ListenerType)

		// Generate xDS virtual hosts and routes for the given HTTPListener,
		// and add them to the xDS route config.
		translator.addRouteToRouteConfig()

	}

}
