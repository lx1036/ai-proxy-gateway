package scheme

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"

	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayapiv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// Add Envoy Gateway types.
	utilruntime.Must(envoygatewayv1alpha1.AddToScheme(scheme))

	// Add Gateway API types.
	utilruntime.Must(gatewayapiv1.Install(scheme))
	utilruntime.Must(gatewayapiv1beta1.Install(scheme))

	// Add CRD kind to known types, experimental conformance test requires this.
	// INFO: 添加CRD类型
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
}

func GetScheme() *runtime.Scheme {
	return scheme
}
