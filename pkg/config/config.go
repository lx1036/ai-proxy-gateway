package config

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/config/mesh"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"k8s.io/klog/v2"
	"os"
)

func ConstructProxyConfig(meshConfigFile, serviceCluster string, concurrency int) (*meshconfig.ProxyConfig, error) {
	contents, err := os.ReadFile(meshConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read mesh config file %v: %v", meshConfigFile, err)
	}
	fileMeshContents := string(contents)

	mc := mesh.DefaultMeshConfig()
	klog.Infof("Apply mesh config from file:\n %v", fileMeshContents)
	fileMesh, err := mesh.ApplyMeshConfig(fileMeshContents, mc)
	if err != nil || fileMesh == nil {
		return nil, fmt.Errorf("failed to unmarshal mesh config from file [%v]: %v", fileMeshContents, err)
	}

	//proxyConfig := mesh.DefaultProxyConfig()
	//if fileMesh.DefaultConfig != nil {
	//	proxyConfig = meshConfig.DefaultConfig
	//}

	return fileMesh.DefaultConfig, nil
}
