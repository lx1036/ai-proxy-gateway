package kube

import (
	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"

	// k8s schema
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	// k8s crd schema
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	// istio schema
	clientextensions "istio.io/client-go/pkg/apis/extensions/v1alpha1"
	clientnetworking "istio.io/client-go/pkg/apis/networking/v1"
	clientnetworkingalpha "istio.io/client-go/pkg/apis/networking/v1alpha3"
	clientnetworkingbeta "istio.io/client-go/pkg/apis/networking/v1beta1"
	clientsecurity "istio.io/client-go/pkg/apis/security/v1"
	clientsecuritybeta "istio.io/client-go/pkg/apis/security/v1beta1"
	clienttelemetry "istio.io/client-go/pkg/apis/telemetry/v1"
	clienttelemetryalpha "istio.io/client-go/pkg/apis/telemetry/v1alpha1"

	// gateway-api schema
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayapialpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
	gatewayapibeta "sigs.k8s.io/gateway-api/apis/v1beta1"
	gatewayx "sigs.k8s.io/gateway-api/apisx/v1alpha1"

	// gateway-api-inference-extension schema
	inferencev1 "sigs.k8s.io/gateway-api-inference-extension/api/v1"
)

// IstioScheme returns a scheme will all known Istio-related types added
var (
	AIProxyGatewayScheme = aiProxyGatewayScheme()
)

// DefaultRestConfig returns the rest.Config for the given kube config file and context.
func DefaultRestConfig(kubeconfig, configContext string, fns ...func(*rest.Config)) (*rest.Config, error) {
	config, err := BuildClientConfig(kubeconfig, configContext)
	if err != nil {
		return nil, fmt.Errorf("failed to setup client: %v", err)
	}

	for _, fn := range fns {
		fn(config)
	}

	return config, nil
}

// BuildClientConfig builds a client rest config from a kubeconfig filepath and context.
// It overrides the current context with the one provided (empty to use default).
//
// This is a modified version of k8s.io/client-go/tools/clientcmd/BuildConfigFromFlags with the
// difference that it loads default configs if not running in-cluster.
func BuildClientConfig(kubeconfig, context string) (*rest.Config, error) {
	c, err := BuildClientCmd(kubeconfig, context).ClientConfig()
	if err != nil {
		return nil, err
	}
	return SetRestDefaults(c), nil
}

func ConfigLoadingRules(kubeconfig string) *clientcmd.ClientConfigLoadingRules {
	// Config loading rules:
	// 1. kubeconfig if it not empty string
	// 2. Config(s) in KUBECONFIG environment variable
	// 3. In cluster config if running in-cluster
	// 4. Use $HOME/.kube/config
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = kubeconfig
	return loadingRules
}

// BuildClientCmd builds a client cmd config from a kubeconfig filepath and context.
// It overrides the current context with the one provided (empty to use default).
//
// This is a modified version of k8s.io/client-go/tools/clientcmd/BuildConfigFromFlags with the
// difference that it loads default configs if not running in-cluster.
func BuildClientCmd(kubeconfig, context string, overrides ...func(*clientcmd.ConfigOverrides)) clientcmd.ClientConfig {
	loadingRules := ConfigLoadingRules(kubeconfig)
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  context,
	}

	for _, fn := range overrides {
		fn(configOverrides)
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
}

// SetRestDefaults is a helper function that sets default values for the given rest.Config.
// This function is idempotent.
func SetRestDefaults(config *rest.Config) *rest.Config {
	if config.GroupVersion == nil || config.GroupVersion.Empty() {
		config.GroupVersion = &corev1.SchemeGroupVersion
	}
	if len(config.APIPath) == 0 {
		if len(config.GroupVersion.Group) == 0 {
			config.APIPath = "/api"
		} else {
			config.APIPath = "/apis"
		}
	}
	if len(config.ContentType) == 0 {
		//config.ContentType = runtime.ContentTypeJSON

		// Prefer to accept protobuf, but send JSON. This is due to some types (CRDs)
		// not accepting protobuf.
		// If we end up writing many core types in the future we may want to set ContentType to
		// ContentTypeProtobuf only for the core client.
		config.AcceptContentTypes = runtime.ContentTypeProtobuf + "," + runtime.ContentTypeJSON
		config.ContentType = runtime.ContentTypeJSON
	}
	if config.NegotiatedSerializer == nil {
		// This codec factory ensures the resources are not converted. Therefore, resources
		// will not be round-tripped through internal versions. Defaulting does not happen
		// on the client.
		config.NegotiatedSerializer = serializer.NewCodecFactory(AIProxyGatewayScheme).WithoutConversion()
	}
	if len(config.UserAgent) == 0 {
		config.UserAgent = AIProxyGatewayUserAgent()
	}

	return config
}

func aiProxyGatewayScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(kubescheme.AddToScheme(scheme))
	//utilruntime.Must(mcs.AddToScheme(scheme))
	utilruntime.Must(clientnetworking.AddToScheme(scheme))
	utilruntime.Must(clientnetworkingbeta.AddToScheme(scheme))
	utilruntime.Must(clientnetworkingalpha.AddToScheme(scheme))
	utilruntime.Must(clientsecurity.AddToScheme(scheme))
	utilruntime.Must(clientsecuritybeta.AddToScheme(scheme))
	utilruntime.Must(clienttelemetry.AddToScheme(scheme))
	utilruntime.Must(clienttelemetryalpha.AddToScheme(scheme))
	utilruntime.Must(clientextensions.AddToScheme(scheme))
	utilruntime.Must(gatewayapi.Install(scheme))
	utilruntime.Must(gatewayapialpha3.Install(scheme))
	utilruntime.Must(gatewayapibeta.Install(scheme))
	utilruntime.Must(gatewayapiv1.Install(scheme))
	utilruntime.Must(gatewayx.Install(scheme))
	utilruntime.Must(inferencev1.Install(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	return scheme
}

func AIProxyGatewayUserAgent() string {
	return "ai-proxy-gateway/v1.0"
}
