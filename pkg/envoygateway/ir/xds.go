package ir

// Xds
// +k8s:deepcopy-gen=true
type Xds struct {
	HTTP []*HTTPListener `json:"http,omitempty" yaml:"http,omitempty"`

	// EnvoyPatchPolicies is the intermediate representation of the EnvoyPatchPolicy resource
	EnvoyPatchPolicies []*EnvoyPatchPolicy `json:"envoyPatchPolicies,omitempty" yaml:"envoyPatchPolicies,omitempty"`
}

func NewXds() *Xds {

}
