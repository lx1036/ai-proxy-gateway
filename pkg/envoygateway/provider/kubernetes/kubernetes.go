package kubernetes

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/lx1036/gateway/pkg/envoygateway/message"
	"github.com/lx1036/gateway/pkg/envoygateway/scheme"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"

	discoveryv1 "k8s.io/api/discovery/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	gatewayapiv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type Provider struct {
	mgr manager.Manager
}

func NewProvider(ctx context.Context, resources *message.ProviderResources) (*Provider, error) {
	restCfg := ctrl.GetConfigOrDie()
	mgrOpts := manager.Options{
		Scheme:         scheme.GetScheme(),
		MapperProvider: nil,
		Cache: cache.Options{
			// INFO: 去掉 managedFields 字段以 reduce memory usage
			DefaultTransform: cache.TransformStripManagedFields(),
			ByObject: map[client.Object]cache.ByObject{
				// INFO: 只 ListAndWatch envoyproxy labels Pods
				&corev1.Pod{}: {
					Label: labels.SelectorFromSet(scheme.EnvoyAppLabel()),
				},

				// INFO: 为了减少 memory 占用
				&appsv1.Deployment{}: {
					UnsafeDisableDeepCopy: ptr.To(true),
				},
				&appsv1.DaemonSet{}: {
					UnsafeDisableDeepCopy: ptr.To(true),
				},
				&corev1.Node{}: {
					UnsafeDisableDeepCopy: ptr.To(true),
				},
				&corev1.Service{}: {
					UnsafeDisableDeepCopy: ptr.To(true),
				},
				&discoveryv1.EndpointSlice{}: {
					UnsafeDisableDeepCopy: ptr.To(true),
				},
				&corev1.Secret{}: {
					UnsafeDisableDeepCopy: ptr.To(true),
				},
				&corev1.ConfigMap{}: {
					UnsafeDisableDeepCopy: ptr.To(true),
				},
				// 目前 k8s 里安装的是 v1beta1 的 ReferenceGrant，先临时用 v1beta1 版本的
				&gatewayapiv1beta1.ReferenceGrant{}: {
					UnsafeDisableDeepCopy: ptr.To(true),
				},
			},
		},
		NewCache: nil,
		Client: client.Options{
			Cache: &client.CacheOptions{
				Unstructured: true,
			},
		},
		NewClient:                           nil,
		Logger:                              ctrl.LoggerFrom(ctx),
		LeaderElection:                      false,
		LeaderElectionResourceLock:          "",
		LeaderElectionNamespace:             "",
		LeaderElectionID:                    "",
		LeaderElectionConfig:                nil,
		LeaderElectionReleaseOnCancel:       false,
		LeaderElectionLabels:                nil,
		LeaderElectionResourceLockInterface: nil,
		LeaseDuration:                       nil,
		RenewDeadline:                       nil,
		RetryPeriod:                         nil,
		HealthProbeBindAddress:              "",
		ReadinessEndpointName:               "",
		LivenessEndpointName:                "",
		PprofBindAddress:                    "",
		WebhookServer:                       nil,
		BaseContext:                         nil,
		EventBroadcaster:                    nil,
		GracefulShutdownTimeout:             nil,
		Controller:                          config.Controller{},
	}
	// INFO: 临时用 logr.New(log.NullLogSink{})，后续再看下问题
	log.SetLogger(logr.New(log.NullLogSink{}))

	mgr, err := ctrl.NewManager(restCfg, mgrOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	err = mgr.AddHealthzCheck("healthz", healthz.Ping)
	if err != nil {
		return nil, fmt.Errorf("add healthz check: %w", err)
	}

	// INFO: 这里写 WaitForCacheSync()，这样不用在 start 或者 外层写了。
	err = mgr.AddReadyzCheck("cache-sync", func(req *http.Request) error {
		if !mgr.GetCache().WaitForCacheSync(ctx) {
			return fmt.Errorf("cache not sync yet")
		}

		klog.Infof("cache is synced")
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("add readyz check: %w", err)
	}

	err = NewGatewayAPIReconciler(ctx, mgr, resources)
	if err != nil {
		return nil, fmt.Errorf("create gateway api reconciler: %w", err)
	}

	klog.Infof("new kubernetes provider")
	return &Provider{
		mgr: mgr,
	}, nil
}

func (provider *Provider) Start(ctx context.Context) error {
	errChan := make(chan error)
	go func() {
		err := provider.mgr.Start(ctx)
		errChan <- err
	}()

	select {
	case <-ctx.Done():
		klog.Infof("kubernetes provider is terminating")
		return nil
	case err := <-errChan:
		klog.Errorf("start manager error: %v", err)
		return err
	}
}
