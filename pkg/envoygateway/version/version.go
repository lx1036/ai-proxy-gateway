package version

import (
	"fmt"
	"io"
	"runtime"
	"encoding/json"
	"sigs.k8s.io/yaml"
)

type Info struct {
	EnvoyGatewayVersion string `json:"envoyGatewayVersion"`
	GatewayAPIVersion   string `json:"gatewayAPIVersion"`
	EnvoyProxyVersion   string `json:"envoyProxyVersion"`
	GitCommitID         string `json:"gitCommitID"`
	GolangVersion       string `json:"golangVersion"`
}

func Get() Info {
	return Info{
		EnvoyGatewayVersion: "1.0.0",
		GatewayAPIVersion:   "v1",
		EnvoyProxyVersion:   "v1alpha1",
		GitCommitID:         "123abc",
		GolangVersion:       runtime.Version(),
	}
}

func Print(w io.Writer, format string) error {
	v := Get()
	switch format {
	case "json":
		if marshalled, err := json.MarshalIndent(v, "", "  "); err == nil {
			_, _ = fmt.Fprintln(w, string(marshalled))
		}
	case "yaml":
		if marshalled, err := yaml.Marshal(v); err == nil {
			_, _ = fmt.Fprintln(w, string(marshalled))
		}
	default:
		_, _ = fmt.Fprintf(w, "ENVOY_GATEWAY_VERSION: %s\n", v.EnvoyGatewayVersion)
		_, _ = fmt.Fprintf(w, "ENVOY_PROXY_VERSION: %s\n", v.EnvoyProxyVersion)
		_, _ = fmt.Fprintf(w, "GATEWAYAPI_VERSION: %s\n", v.GatewayAPIVersion)
		_, _ = fmt.Fprintf(w, "GIT_COMMIT_ID: %s\n", v.GitCommitID)
		_, _ = fmt.Fprintf(w, "GOLANG_VERSION: %s\n", v.GolangVersion)
	}

	return nil
}
