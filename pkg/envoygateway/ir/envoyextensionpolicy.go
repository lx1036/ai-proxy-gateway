package ir

// EnvoyExtensionFeatures holds the information associated with the Envoy Extension Policy.
// +k8s:deepcopy-gen=true
type EnvoyExtensionFeatures struct {
	// External Processing extensions
	ExtProcs []ExtProc `json:"extProcs,omitempty" yaml:"extProcs,omitempty"`
	// Wasm extensions
	//Wasms []Wasm `json:"wasms,omitempty" yaml:"wasms,omitempty"`
	// Lua extensions
	//Luas []Lua `json:"luas,omitempty" yaml:"luas,omitempty"`
}
