package scheme





// EnvoyAppLabel returns the labels used for all Envoy resources.
func EnvoyAppLabel() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "envoy",
		"app.kubernetes.io/component":  "proxy",
		"app.kubernetes.io/managed-by": "envoy-gateway",
	}
}



