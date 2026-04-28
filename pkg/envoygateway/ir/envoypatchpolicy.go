package ir

import apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

// EnvoyPatchPolicy defines the intermediate representation of the EnvoyPatchPolicy resource.
// +k8s:deepcopy-gen=true
type EnvoyPatchPolicy struct {
	// JSONPatches are the JSON Patches that
	// are to be applied to generated Xds linked to the gateway.
	JSONPatches []*JSONPatchConfig `json:"jsonPatches,omitempty" yaml:"jsonPatches,omitempty"`
}

// JSONPatchConfig defines the configuration for patching a Envoy xDS Resource
// using JSONPatch semantics
// +k8s:deepcopy-gen=true
type JSONPatchConfig struct {
	// Type is the typed URL of the Envoy xDS Resource
	Type string `json:"type" yaml:"type"`
	// Name is the name of the resource
	Name string `json:"name" yaml:"name"`
	// Patch defines the JSON Patch Operation
	Operation JSONPatchOperation `json:"operation" yaml:"operation"`
}

type JSONPatchOp string

const (
	JSONPatchOpAdd     JSONPatchOp = "add"
	JSONPatchOpRemove  JSONPatchOp = "remove"
	JSONPatchOpReplace JSONPatchOp = "replace"
	JSONPatchOpCopy    JSONPatchOp = "copy"
	JSONPatchOpMove    JSONPatchOp = "move"
	JSONPatchOpTest    JSONPatchOp = "test"
)

// JSONPatchOperation defines the JSON Patch Operation as defined in
// https://datatracker.ietf.org/doc/html/rfc6902
// +k8s:deepcopy-gen=true
type JSONPatchOperation struct {
	// Op is the type of operation to perform
	Op JSONPatchOp `json:"op" yaml:"op"`
	// Path is the location of the target document/field where the operation will be performed
	// Refer to https://datatracker.ietf.org/doc/html/rfc6901 for more details.
	// +optional
	Path *string `json:"path,omitempty"  yaml:"path,omitempty"`
	// JSONPath specifies the locations of the target document/field where the operation will be performed
	// Refer to https://datatracker.ietf.org/doc/rfc9535/ for more details.
	// +optional
	JSONPath *string `json:"jsonPath,omitempty"  yaml:"jsonPath,omitempty"`
	// From is the source location of the value to be copied or moved. Only valid
	// for move or copy operations
	// Refer to https://datatracker.ietf.org/doc/html/rfc6901 for more details.
	// +optional
	From *string `json:"from,omitempty" yaml:"from,omitempty"`
	// Value is the new value of the path location.
	Value *apiextensionsv1.JSON `json:"value,omitempty" yaml:"value,omitempty"`
}
