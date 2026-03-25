package config

import (
	v1 "github.com/lx1036/gateway/pkg/epp/apis/v1"
	"sigs.k8s.io/yaml"
)

func LoadConfig(configBytes []byte) (*v1.EndpointPickerConfig, error) {

	var eppConfig v1.EndpointPickerConfig
	if err := yaml.Unmarshal(configBytes, &eppConfig); err != nil {
		return nil, err
	}

}
