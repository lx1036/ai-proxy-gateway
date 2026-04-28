package gatewayapi

import (
	"context"
	"github.com/lx1036/gateway/pkg/envoygateway/gatewayapi/resource"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/message"
	"github.com/telepresenceio/watchable"
	"k8s.io/apimachinery/pkg/types"
	"maps"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sync"

	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
)

const (
	OwningGatewayNamespaceLabel = "gateway.envoyproxy.io/owning-gateway-namespace"

	OwningGatewayNameLabel = "gateway.envoyproxy.io/owning-gateway-name"
)

type Translator struct {
	// Goroutine synchronization
	done sync.WaitGroup

	ProviderResources *message.ProviderResources

	InfraIR *message.InfraIR
	XdsIR   *message.XdsIR
}

func NewTranslator(infraIR *message.InfraIR, xdsIR *message.XdsIR) (*Translator, error) {
	return &Translator{
		InfraIR: infraIR,
		XdsIR:   xdsIR,
	}, nil
}

func (translator *Translator) Start(ctx context.Context) error {

	sub := translator.ProviderResources.GatewayAPIResources.Subscribe(ctx)
	go message.HandleSubscription(sub, func(update watchable.Update[string, *message.GatewayAPIResources]) {

		resources := *update.Value
		for _, res := range resources {

			//
			result := translator.Translate(res)

			// INFO: 1. publish infraIR
			for gatewayKey, infra := range result.InfraIR {

				translator.InfraIR.Store(gatewayKey, infra)
			}

			// INFO: 2. publish xdsIR
			for gatewayKey, xds := range result.XdsIR {

				translator.XdsIR.Store(gatewayKey, xds)
			}

		}

	})

	return nil
}

func (translator *Translator) Translate(res *message.GatewayAPIResource) *TranslateResult {

	acceptedGateways, failedGateways := translator.GetRelevantGateways(res)

	xdsIR, infraIR := translator.InitIRs(acceptedGateways, failedGateways)

	// Process all relevant HTTPRoutes.
	httpRoutes := translator.ProcessHTTPRoutes(res)

	// Add both accepted and failed gateways to the result because we need to update the status of all gateways.
	allGateways := make([]*GatewayContext, 0, len(acceptedGateways)+len(failedGateways))
	allGateways = append(allGateways, acceptedGateways...)
	allGateways = append(allGateways, failedGateways...)

	// Process EnvoyExtensionPolicies
	envoyExtensionPolicies := translator.ProcessEnvoyExtensionPolicies()

	// Process EnvoyPatchPolicies
	translator.ProcessEnvoyPatchPolicies(res.EnvoyPatchPolicies, xdsIR)

	return NewTranslateResult(allGateways, httpRoutes, envoyExtensionPolicies, xdsIR, infraIR)
}

func (translator *Translator) InitIRs(acceptedGateways, failedGateways []*GatewayContext) (map[string]*ir.Xds, map[string]*ir.Infra) {

	xdsIR := make(map[string]*ir.Xds)
	infraIR := make(map[string]*ir.Infra)
	for _, gateway := range acceptedGateways {
		irKey, gwXdsIR, gwInfraIR := translator.buildIR(gateway)
		xdsIR[irKey] = gwXdsIR
		infraIR[irKey] = gwInfraIR
	}

	for _, gateway := range failedGateways {
		irKey, gwXdsIR, gwInfraIR := translator.buildIR(gateway)
		xdsIR[irKey] = gwXdsIR
		infraIR[irKey] = gwInfraIR
	}

	return xdsIR, infraIR
}

func (translator *Translator) buildIR(gateway *GatewayContext) (string, *ir.Xds, *ir.Infra) {
	gatewayInfraIR := ir.NewInfra()
	gatewayXdsIR := ir.NewXds()

	irKey := types.NamespacedName{
		Namespace: gateway.Namespace,
		Name:      gateway.Name,
	}.String()
	gatewayInfraIR.Proxy.Name = irKey
	gatewayInfraIR.Proxy.Namespace = ""

	labels := make(map[string]string)
	if gateway.Spec.Infrastructure != nil {
		for key, value := range gateway.Spec.Infrastructure.Labels {
			labels[string(key)] = string(value)
		}
	}

	maps.Copy(labels, map[string]string{
		OwningGatewayNamespaceLabel: gateway.Namespace,
		OwningGatewayNameLabel:      gateway.Name,
	})

	gatewayInfraIR.Proxy.Metadata.Labels = labels

	return irKey, gatewayXdsIR, gatewayInfraIR
}

func (translator *Translator) Close() {

}

type TranslateResult struct {
	resource.Resources
	XdsIR   map[string]*ir.Xds   `json:"xdsIR" yaml:"xdsIR"`
	InfraIR map[string]*ir.Infra `json:"infraIR" yaml:"infraIR"`
}

func NewTranslateResult(
	gateways []*GatewayContext,
	httpRoutes []*HTTPRouteContext,
	envoyExtensionPolicies []*envoygatewayv1alpha1.EnvoyExtensionPolicy,
	xdsIR map[string]*ir.Xds,
	infraIR map[string]*ir.Infra,
) *TranslateResult {
	translateResult := &TranslateResult{
		XdsIR:   xdsIR,
		InfraIR: infraIR,
	}

	if n := len(gateways); n > 0 {
		translateResult.Gateways = make([]*gatewayapiv1.Gateway, n)
		for i, gateway := range gateways {
			translateResult.Gateways[i] = gateway.Gateway
		}
	}

	if n := len(httpRoutes); n > 0 {
		translateResult.HTTPRoutes = make([]*gatewayapiv1.HTTPRoute, n)
		for i, httpRoute := range httpRoutes {
			translateResult.HTTPRoutes[i] = httpRoute.HTTPRoute
		}
	}

	if len(envoyExtensionPolicies) > 0 {
		translateResult.EnvoyExtensionPolicies = envoyExtensionPolicies
	}

	return translateResult
}
