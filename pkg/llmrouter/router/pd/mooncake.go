package pd


// NewMoonCakeConnector creates a new mooncake connector
// MooncakeConnector in vllm-ascend behaves similar as NIXL, so we reuse nixl implementation.
func NewMoonCakeConnector() KVConnector {
	return &NIXLConnector{
		name: "mooncake",
	}
}

