package ir

// Xds
// +k8s:deepcopy-gen=true
type Xds struct {
	HTTP []*HTTPListener `json:"http,omitempty" yaml:"http,omitempty"`
}

func NewXds() *Xds {

}
