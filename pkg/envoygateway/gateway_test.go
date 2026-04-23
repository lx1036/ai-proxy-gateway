package envoygateway

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	leaderworkersetv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
	"testing"
)

/**
(vllm) (⎈|zhisuan-integration-dev:N/A)➜  ai-proxy-gateway git:(main) ✗ kubectl get pods --field-selector spec.nodeName=gpu-2-26e60 -o wide
NAME                   READY   STATUS    RESTARTS   AGE    IP              NODE          NOMINATED NODE   READINESS GATES
available-node-test1   1/1     Running   0          16d    10.243.20.155   gpu-2-26e60   <none>           <none>
az-node-test           1/1     Running   0          48d    10.243.20.108   gpu-2-26e60   <none>           <none>
file-transfer-pod      1/1     Running   0          28d    10.243.20.117   gpu-2-26e60   <none>           <none>
fraction-pod           1/1     Running   0          125d   10.243.20.99    gpu-2-26e60   <none>           <none>
fraction-pod2          1/1     Running   0          125d   10.243.20.104   gpu-2-26e60   <none>           <none>
fraction-pod3          1/1     Running   0          125d   10.243.20.110   gpu-2-26e60   <none>           <none>
localpv-test1          1/1     Running   0          74d    10.243.20.82    gpu-2-26e60   <none>           <none>
localpv-test2          1/1     Running   0          74d    10.243.20.57    gpu-2-26e60   <none>           <none>
localpv-test3          1/1     Running   0          73d    10.243.20.94    gpu-2-26e60   <none>           <none>
no-local-path-test     1/1     Running   0          16d    10.243.20.154   gpu-2-26e60   <none>           <none>
normal-test            1/1     Running   0          41d    10.243.20.133   gpu-2-26e60   <none>           <none>
nsenter-1msv59         1/1     Running   0          100d   10.243.24.4     gpu-2-26e60   <none>           <none>
nsenter-dp0o32         1/1     Running   0          101d   10.243.24.4     gpu-2-26e60   <none>           <none>
nsenter-ok9fe1         1/1     Running   0          100d   10.243.24.4     gpu-2-26e60   <none>           <none>
nsenter-ywm5ky         1/1     Running   0          50d    10.243.24.4     gpu-2-26e60   <none>           <none>

(vllm) (⎈|zhisuan-integration-dev:N/A)➜  ai-proxy-gateway git:(main) ✗ kubectl get gateway -A --field-selector spec.gatewayClassName=envoygateway-liuxiang1
Error from server (BadRequest): Unable to find "gateway.networking.k8s.io/v1, Resource=gateways" that match label selector "", field selector "spec.gatewayClassName=envoygateway-liuxiang1": field label not supported: spec.gatewayClassName
(vllm) (⎈|zhisuan-integration-dev:N/A)➜  ai-proxy-gateway git:(main) ✗


INFO: gateway 不支持 --field-selector spec.gatewayClassName，所以只能 mgr.GetFieldIndexer().IndexField() 来实现
*/

const (
	classGatewayIndex = "gatewayIndex"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(leaderworkersetv1.AddToScheme(scheme))
	utilruntime.Must(gwapiv1.Install(scheme))
}

func TestName(test *testing.T) {
	cfg := ctrl.GetConfigOrDie()
	// informer watch 并从本地 indexer cache 中获取
	mgr, _ := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})

	ctx := ctrl.SetupSignalHandler()

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwapiv1.Gateway{}, classGatewayIndex, gatewayIndexFunc); err != nil {
		klog.Errorf("failed to create index: %v", err)
		return
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			klog.Fatalf("mgr start error: %v", err)
		}
	}()

	if ok := mgr.GetCache().WaitForCacheSync(ctx); !ok {
		klog.Fatalf("failed to wait for caches to sync")
	}

	// INFO: 加上 field selector

	gatewayClassName := "envoygateway-liuxiang1"
	gatewayList := &gwapiv1.GatewayList{}
	if err := mgr.GetClient().List(ctx, gatewayList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(classGatewayIndex, gatewayClassName),
	}); err != nil {
		klog.Errorf("failed to list gateways for GatewayClass %s: %v", gatewayClassName, err)
		return
	}

	for _, item := range gatewayList.Items {
		klog.Infof("gateway: %s/%s", item.Namespace, item.Name)
		// gateway: envoy-gateway-system/envoygateway-liuxiang1
	}

	<-ctx.Done()
}

func gatewayIndexFunc(rawObj client.Object) []string {
	gateway := rawObj.(*gwapiv1.Gateway)
	return []string{string(gateway.Spec.GatewayClassName)} // INFO: gateway.Spec.GatewayClassName 要等于 managedGC.Name, @see gatewayAPIReconciler.processGateways()
}
