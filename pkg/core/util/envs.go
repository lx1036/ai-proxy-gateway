package util

import "istio.io/istio/pkg/env"

var (
	PodNamespace = env.RegisterStringVar("POD_NAMESPACE", "higress-system", "").Get()
	PodName      = env.RegisterStringVar("POD_NAME", "", "").Get()
	GatewayName  = env.RegisterStringVar("GATEWAY_NAME", "higress-gateway", "").Get()
	// Revision is the value of the Istio control plane revision, e.g. "canary",
	// and is the value used by the "istio.io/rev" label.
	Revision              = env.Register("REVISION", "", "").Get()
	McpServerWasmImageUrl = env.RegisterStringVar("MCP_SERVER_WASM_IMAGE_URL", "oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/mcp-server/all-in-one:1.0.0", "").Get()
)
