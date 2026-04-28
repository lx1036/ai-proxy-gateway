package gatewayapi

import (
	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/scheme"
)

func (translator *Translator) ProcessEnvoyExtensionPolicies(
	envoyExtensionPolicies []*envoygatewayv1alpha1.EnvoyExtensionPolicy,
	xdsIR map[string]*ir.Infra,
) []*envoygatewayv1alpha1.EnvoyExtensionPolicy {

	for _, currPolicy := range envoyExtensionPolicies {

		// TODO: 先简化逻辑，只考虑 spec.targetRefs 字段
		for _, targetRef := range currPolicy.Spec.PolicyTargetReferences.TargetRefs {
			// TODO: 先不考虑 .SectionName 字段
			if targetRef.Kind == scheme.KindHTTPRoute {

				translator.processEnvoyExtensionPolicyForRoute(targetRef, xdsIR)
			}
		}

	}

	for _, currPolicy := range envoyExtensionPolicies {

		// TODO: 先简化逻辑，只考虑 spec.targetRefs 字段
		for _, targetRef := range currPolicy.Spec.PolicyTargetReferences.TargetRefs {
			// TODO: 先不考虑 .SectionName 字段
			if targetRef.Kind == scheme.KindGateway {
				translator.processEnvoyExtensionPolicyForGateway(targetRef, xdsIR)
			}
		}
	}

}

func (translator *Translator) processEnvoyExtensionPolicyForRoute() {
	translator.translateEnvoyExtensionPolicyForRoute()
}

func (translator *Translator) translateEnvoyExtensionPolicyForRoute() {
	for _, p := range parentRefs {
		// TODO: 1. lua
		// TODO: 2. wasm

		// INFO: 3. extproc
		translator.buildExtProcs()
	}
}

func (translator *Translator) buildExtProcs(policy *envoygatewayv1alpha1.EnvoyExtensionPolicy) []ir.ExtProc {

	extProcIRList := make([]ir.ExtProc, 0, len(policy.Spec.ExtProc))

	for _, extProc := range policy.Spec.ExtProc {
		extProcIR, err := translator.buildExtProc(extProc)
		if err != nil {

			continue
		}

		extProcIRList = append(extProcIRList, *extProcIR)

	}

	return extProcIRList
}

func (translator *Translator) buildExtProc(extProc envoygatewayv1alpha1.ExtProc) (*ir.ExtProc, error) {

	routeDestination, err := translator.translateExtServiceBackendRefs(extProc)

	extProcIR := &ir.ExtProc{
		Name:        name,
		Destination: *routeDestination,
		Traffic:     traffic,
		Authority:   authority,
	}

	return extProcIR, nil
}

func (translator *Translator) processEnvoyExtensionPolicyForGateway() {

}
