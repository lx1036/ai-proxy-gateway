package ir

import (
	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
)

type ExtProcBodyProcessingMode envoygatewayv1alpha1.ExtProcBodyProcessingMode

const (
	// ExtProcBodyStreamed sets the streamed body processing mode
	ExtProcBodyStreamed = ExtProcBodyProcessingMode(envoygatewayv1alpha1.StreamedExtProcBodyProcessingMode)
	// ExtProcBodyBuffered sets the buffered body processing mode
	ExtProcBodyBuffered = ExtProcBodyProcessingMode(envoygatewayv1alpha1.BufferedExtProcBodyProcessingMode)
	// ExtProcBodyBufferedPartial sets the partial buffered body processing mode
	ExtProcBodyBufferedPartial = ExtProcBodyProcessingMode(envoygatewayv1alpha1.BufferedPartialExtBodyHeaderProcessingMode)
	// ExtProcBodyFullDuplexStreamed sets the full duplex streamed processing mode
	ExtProcBodyFullDuplexStreamed = ExtProcBodyProcessingMode(envoygatewayv1alpha1.FullDuplexStreamedExtBodyProcessingMode)
)

// ExtProc holds the information associated with the ExtProc extensions.
// +k8s:deepcopy-gen=true
type ExtProc struct {
	Name string `json:"name" yaml:"name"`

	// Authority is the hostname:port of the HTTP External Processing service.
	Authority string `json:"authority" yaml:"authority"`

	// Destination defines the destination for the gRPC External Processing service.
	Destination RouteDestination `json:"destination" yaml:"destination"`

	// RequestHeaderProcessing Defines if request headers are processed
	RequestHeaderProcessing bool `json:"requestHeaderProcessing,omitempty" yaml:"requestHeaderProcessing,omitempty"`

	// RequestBodyProcessingMode Defines request body processing
	RequestBodyProcessingMode *ExtProcBodyProcessingMode `json:"requestBodyProcessingMode,omitempty" yaml:"requestBodyProcessingMode,omitempty"`

	// ResponseHeaderProcessingMode Defines if response headers are processed
	ResponseHeaderProcessing bool `json:"responseHeaderProcessing,omitempty" yaml:"responseHeaderProcessing,omitempty"`

	// ResponseBodyProcessingMode Defines response body processing
	ResponseBodyProcessingMode *ExtProcBodyProcessingMode `json:"responseBodyProcessingMode,omitempty" yaml:"responseBodyProcessingMode,omitempty"`
}
