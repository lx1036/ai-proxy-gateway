package extensionserver

import (
	"fmt"
	egextension "github.com/envoyproxy/gateway/proto/extension"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	gwaiev1 "sigs.k8s.io/gateway-api-inference-extension/api/v1"

	"encoding/json"
)

const (
	// EndpointPickerHeaderKey is the header key used to specify the target backend endpoint.
	// This is the default header name in the reference implementation:
	// https://github.com/kubernetes-sigs/gateway-api-inference-extension/blob/2b5b337b45c3289e5f9367b2c19deef021722fcd/pkg/epp/server/runserver.go#L63
	EndpointPickerHeaderKey = "x-gateway-destination-endpoint"

	// defaultEndpointPickerPort is the default port for Gateway API Inference Extension endpoint picker services.
	// This port is commonly used by EPP (Endpoint Picker Protocol) services as defined in the
	// Gateway API Inference Extension specification and examples.
	// See: https://gateway-api-inference-extension.sigs.k8s.io/
	defaultEndpointPickerPort = 9002

	// processingBodyModeAnnotation is the annotation key for configuring processing body mode
	processingBodyModeAnnotation = "aigateway.envoyproxy.io/processing-body-mode"
	// allowModeOverrideAnnotation is the annotation key for configuring allow mode override
	allowModeOverrideAnnotation = "aigateway.envoyproxy.io/allow-mode-override"

	// InternalEndpointMetadataNamespace is the namespace used for the dynamic metadata for internal use.
	InternalEndpointMetadataNamespace = "aigateway.envoy.io"

	// internalMetadataInferencePoolKey is the key used to store the inference pool metadata.
	// This is only used within the extension server for InferencePool cluster identification.
	internalMetadataInferencePoolKey = "per_route_rule_inference_pool"
)

func (s *Server) constructInferencePoolsFrom(extensionResources []*egextension.ExtensionResource) []*gwaiev1.InferencePool {
	// Parse InferencePool resources from BackendExtensionResources.
	// BackendExtensionResources contains unstructured Kubernetes resources that were
	// referenced in the AIGatewayRoute's BackendRefs with non-empty Group and Kind fields.
	var inferencePools []*gwaiev1.InferencePool
	for _, resource := range extensionResources {
		// Unmarshal the unstructured bytes to get the Kubernetes resource.
		// The resource is stored as JSON bytes in the extension context.
		var unstructuredObj unstructured.Unstructured

		// 这里没用 encoding/json 包
		if err := json.Unmarshal(resource.UnstructuredBytes, &unstructuredObj); err != nil {
			klog.ErrorS(err, "failed to unmarshal extension resource", "resource_size", len(resource.UnstructuredBytes))
			continue
		}

		// Check if this is an InferencePool resource from the Gateway API Inference Extension.
		// We only process InferencePool resources; other extension resources are ignored.
		if unstructuredObj.GetAPIVersion() == "inference.networking.k8s.io/v1" &&
			unstructuredObj.GetKind() == "InferencePool" {
			// Convert unstructured object to strongly-typed InferencePool.
			// This allows us to access the InferencePool's spec fields safely.
			var pool gwaiev1.InferencePool
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &pool); err != nil {
				klog.ErrorS(err, "failed to convert unstructured to InferencePool",
					"name", unstructuredObj.GetName(), "namespace", unstructuredObj.GetNamespace())
				continue
			}
			inferencePools = append(inferencePools, &pool)
		}
	}

	return inferencePools
}

// buildMetadataForInferencePool adds InferencePool metadata to the cluster for reference by other components.
// encoded as a string in the format: "namespace/name/serviceName/port".
func buildEPPMetadataForCluster(cluster *clusterv3.Cluster, inferencePool *gwaiev1.InferencePool) {
	// Initialize cluster metadata structure if not present.
	buildEPPMetadata(cluster.Metadata, inferencePool)
}

// buildEPPMetadata adds InferencePool metadata to the given metadata structure.
func buildEPPMetadata(metadata *corev3.Metadata, inferencePool *gwaiev1.InferencePool) {
	// Initialize cluster metadata structure if not present.
	if metadata == nil {
		metadata = &corev3.Metadata{}
	}
	if metadata.FilterMetadata == nil {
		metadata.FilterMetadata = make(map[string]*structpb.Struct)
	}

	// Get or create the internal metadata namespace for AI Gateway.
	m, ok := metadata.FilterMetadata[InternalEndpointMetadataNamespace]
	if !ok {
		m = &structpb.Struct{}
		metadata.FilterMetadata[InternalEndpointMetadataNamespace] = m
	}
	if m.Fields == nil {
		m.Fields = make(map[string]*structpb.Value)
	}

	// Read processing body mode from annotations, default to "duplex" (FULL_DUPLEX_STREAMED)
	processingBodyMode := getProcessingBodyModeStringFromAnnotations(inferencePool)
	// Read allow mode override from annotations, default to false
	allowModeOverride := getAllowModeOverrideStringFromAnnotations(inferencePool)

	// Store InferencePool reference as metadata for later retrieval.
	// The reference includes all information needed to build EPP clusters and filters.
	m.Fields[internalMetadataInferencePoolKey] = structpb.NewStringValue(
		clusterRefInferencePool(
			inferencePool.Namespace,
			inferencePool.Name,
			string(inferencePool.Spec.EndpointPickerRef.Name),
			portForInferencePool(inferencePool),
			processingBodyMode,
			allowModeOverride,
		),
	)
}

// clusterRefInferencePool generates a unique reference for an InferencePool cluster.
func clusterRefInferencePool(namespace, name, serviceName string, servicePort uint32, bodyMode string, allowModeOverride string) string {
	return fmt.Sprintf("%s/%s/%s/%d/%s/%s", namespace, name, serviceName, servicePort, bodyMode, allowModeOverride)
}

// getProcessingBodyModeStringFromAnnotations reads the processing body mode from InferencePool annotations.
func getProcessingBodyModeStringFromAnnotations(pool *gwaiev1.InferencePool) string {
	annotations := pool.GetAnnotations()
	if annotations == nil {
		return "duplex" // default to duplex
	}

	mode, exists := annotations[processingBodyModeAnnotation]
	if !exists {
		return "duplex" // default to duplex
	}

	return mode
}

// getAllowModeOverrideStringFromAnnotations reads the allow mode override setting from InferencePool annotations.
func getAllowModeOverrideStringFromAnnotations(pool *gwaiev1.InferencePool) string {
	annotations := pool.GetAnnotations()
	if annotations == nil {
		return "false" // default to false
	}

	value, exists := annotations[allowModeOverrideAnnotation]
	if !exists {
		return "false" // default to false
	}

	return value
}

// portForInferencePool returns the port number for the given InferencePool.
func portForInferencePool(pool *gwaiev1.InferencePool) uint32 {
	if p := pool.Spec.EndpointPickerRef.Port; p == nil {
		return defaultEndpointPickerPort
	}
	portNumber := pool.Spec.EndpointPickerRef.Port.Number
	if portNumber < 0 || portNumber > 65535 {
		return defaultEndpointPickerPort // fallback to default port.
	}
	// Safe conversion: portNumber is validated to be in range [0, 65535].
	return uint32(portNumber) // #nosec G1151
}
