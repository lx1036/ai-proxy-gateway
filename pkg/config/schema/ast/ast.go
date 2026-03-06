package ast

import (
	"encoding/json"
	"fmt"
	"github.com/lx1036/gateway/pkg/config/validation"
	"github.com/lx1036/gateway/pkg/util/strcase"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/yaml"
)

type Resource struct {
	Identifier         string   `json:"identifier"`
	Group              string   `json:"group"`
	Version            string   `json:"version"`
	VersionAliases     []string `json:"versionAliases"`
	Kind               string   `json:"kind"`
	Plural             string   `json:"plural"`
	ClusterScoped      bool     `json:"clusterScoped"`
	Builtin            bool     `json:"builtin"`
	Specless           bool     `json:"specless"`
	Synthetic          bool     `json:"synthetic"`
	Proto              string   `json:"proto"`
	ProtoPackage       string   `json:"protoPackage"`
	StatusProto        string   `json:"statusProto"`
	StatusProtoPackage string   `json:"statusProtoPackage"`
	Validate           string   `json:"validate"`
	Description        string   `json:"description"`
}

type Metadata struct {
	Resources []*Resource `json:"resources"`
}

func Parse(yamlText string) (*Metadata, error) {
	var s Metadata
	err := yaml.Unmarshal([]byte(yamlText), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UnmarshalJSON implements json.Unmarshaler
func (m *Metadata) UnmarshalJSON(data []byte) error {
	var in struct {
		Resources []*Resource `json:"resources"`
	}

	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}

	m.Resources = in.Resources
	seen := sets.New[string]()
	// Process resources.
	for i, r := range m.Resources {
		if r.Validate == "" {
			validateFn := "Validate" + asResourceVariableName(r.Kind)
			if !validation.IsValidateFunc(validateFn) {
				validateFn = "validation.EmptyValidate"
			} else {
				if r.Kind == "EnvoyFilter" {
					validateFn = "envoyfilter." + validateFn
				} else {
					validateFn = "validation." + validateFn
				}
			}
			m.Resources[i].Validate = validateFn
		}
		if r.Identifier == "" {
			r.Identifier = r.Kind
		}

		if seen.Has(r.Identifier) {
			return fmt.Errorf("identifier %q already registered, set a unique identifier", r.Identifier)
		} else {
			seen.Insert(r.Identifier)
		}
	}

	return nil
}

func asResourceVariableName(n string) string {
	return strcase.CamelCase(n)
}
