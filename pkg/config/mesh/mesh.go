package mesh

import (
	"github.com/lx1036/gateway/pkg/util/protomarshal"
	"google.golang.org/protobuf/types/known/durationpb"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/api/networking/v1alpha3"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"

	"time"
)

// DefaultMeshConfig returns the default mesh config.
// This is merged with values from the mesh config map.
func DefaultMeshConfig() *meshconfig.MeshConfig {
	return &meshconfig.MeshConfig{
		EnableTracing:               true,
		AccessLogFile:               "",
		AccessLogEncoding:           meshconfig.MeshConfig_TEXT,
		AccessLogFormat:             "",
		EnableEnvoyAccessLogService: false,
		ProtocolDetectionTimeout:    durationpb.New(0),
		IngressService:              "ai-proxy-ingressgateway",
		IngressControllerMode:       meshconfig.MeshConfig_STRICT,
		IngressClass:                "ai-proxy-gateway",
		TrustDomain:                 "cluster.local",
		TrustDomainAliases:          []string{},
		EnableAutoMtls:              wrappers.Bool(true),
		OutboundTrafficPolicy:       &meshconfig.MeshConfig_OutboundTrafficPolicy{Mode: meshconfig.MeshConfig_OutboundTrafficPolicy_ALLOW_ANY},
		InboundTrafficPolicy:        &meshconfig.MeshConfig_InboundTrafficPolicy{Mode: meshconfig.MeshConfig_InboundTrafficPolicy_PASSTHROUGH},
		LocalityLbSetting: &v1alpha3.LocalityLoadBalancerSetting{
			Enabled: wrappers.Bool(true),
		},
		Certificates:  []*meshconfig.Certificate{},
		DefaultConfig: DefaultProxyConfig(),

		RootNamespace:                  "ai-gateway-system",
		ProxyListenPort:                15001,
		ProxyInboundListenPort:         15006,
		ConnectTimeout:                 durationpb.New(10 * time.Second),
		DefaultServiceExportTo:         []string{"*"},
		DefaultVirtualServiceExportTo:  []string{"*"},
		DefaultDestinationRuleExportTo: []string{"*"},
		// DnsRefreshRate is only used when DNS requests fail (NXDOMAIN or SERVFAIL). For success, the TTL
		// will be used.
		// https://datatracker.ietf.org/doc/html/rfc2308#section-3 defines how negative DNS results should handle TTLs,
		// but Envoy does not respect this (https://github.com/envoyproxy/envoy/issues/20885).
		// To counter this, we bump up the default to 60s to avoid overloading DNS servers.
		DnsRefreshRate:  durationpb.New(60 * time.Second),
		ServiceSettings: make([]*meshconfig.MeshConfig_ServiceSettings, 0),

		EnablePrometheusMerge: wrappers.Bool(true),
		DefaultProviders:      &meshconfig.MeshConfig_DefaultProviders{},
		ExtensionProviders: []*meshconfig.MeshConfig_ExtensionProvider{
			{
				Name: "prometheus",
				Provider: &meshconfig.MeshConfig_ExtensionProvider_Prometheus{
					Prometheus: &meshconfig.MeshConfig_ExtensionProvider_PrometheusMetricsProvider{},
				},
			},
			{
				Name: "stackdriver",
				Provider: &meshconfig.MeshConfig_ExtensionProvider_Stackdriver{
					Stackdriver: &meshconfig.MeshConfig_ExtensionProvider_StackdriverProvider{},
				},
			},
			{
				Name: "envoy",
				Provider: &meshconfig.MeshConfig_ExtensionProvider_EnvoyFileAccessLog{
					EnvoyFileAccessLog: &meshconfig.MeshConfig_ExtensionProvider_EnvoyFileAccessLogProvider{
						Path: "/dev/stdout",
					},
				},
			},
		},
	}
}

func DefaultProxyConfig() *meshconfig.ProxyConfig {
	return &meshconfig.ProxyConfig{
		ConfigPath:               "/etc/istio/proxy",
		ClusterName:              &meshconfig.ProxyConfig_ServiceCluster{ServiceCluster: "ai-proxy-gateway"},
		DrainDuration:            durationpb.New(45 * time.Second),
		TerminationDrainDuration: durationpb.New(5 * time.Second),
		ProxyAdminPort:           15000,
		ControlPlaneAuthPolicy:   meshconfig.AuthenticationPolicy_MUTUAL_TLS,
		DiscoveryAddress:         "ai-proxy-gateway.ai-gateway-system.svc:15012",

		// Code defaults
		BinaryPath:     "/usr/local/bin/envoy",
		StatNameLength: 189,
		StatusPort:     15020,
	}
}

func ApplyMeshConfig(yaml string, defaultConfig *meshconfig.MeshConfig) (*meshconfig.MeshConfig, error) {
	if err := protomarshal.ApplyYAML(yaml, defaultConfig); err != nil {
		return nil, err
	}

	return defaultConfig, nil
}
