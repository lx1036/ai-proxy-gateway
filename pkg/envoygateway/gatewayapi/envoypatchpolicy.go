package gatewayapi

import (
	"fmt"
	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
)

func (translator *Translator) ProcessEnvoyPatchPolicies(envoyExtensionPolicies []*envoygatewayv1alpha1.EnvoyPatchPolicy, xdsIR map[string]*ir.Xds) {
	for _, envoyPatchPolicy := range envoyExtensionPolicies {
		irKey := fmt.Sprintf("%s/%s", envoyPatchPolicy.Namespace, envoyPatchPolicy.Spec.TargetRef.Name)
		gwXdsIR, ok := xdsIR[irKey]
		if !ok {
			// The TargetRef Gateway is not an accepted Gateway, then skip processing.
			continue
		}

		// Create the IR with the context need to publish the status later
		policyIR := &ir.EnvoyPatchPolicy{}
		policyIR.Name = envoyPatchPolicy.Name
		policyIR.Namespace = envoyPatchPolicy.Namespace
		policyIR.Generation = envoyPatchPolicy.Generation
		policyIR.Status = &envoyPatchPolicy.Status

		// Append the IR
		gwXdsIR.EnvoyPatchPolicies = append(gwXdsIR.EnvoyPatchPolicies, policyIR)

		// Save the patch
		for _, patch := range envoyPatchPolicy.Spec.JSONPatches {
			irPatch := ir.JSONPatchConfig{}
			irPatch.Type = string(patch.Type)
			irPatch.Name = patch.Name
			irPatch.Operation.Op = ir.JSONPatchOp(patch.Operation.Op)
			irPatch.Operation.Path = patch.Operation.Path
			irPatch.Operation.JSONPath = patch.Operation.JSONPath
			irPatch.Operation.From = patch.Operation.From
			irPatch.Operation.Value = patch.Operation.Value

			policyIR.JSONPatches = append(policyIR.JSONPatches, &irPatch)
		}

	}
}
