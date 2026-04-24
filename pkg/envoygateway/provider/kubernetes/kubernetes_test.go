package kubernetes

import (
	"github.com/lx1036/gateway/pkg/envoygateway/message"
	"github.com/telepresenceio/watchable"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	"testing"
)

func TestKubernetesProvider(test *testing.T) {
	ctx := ctrl.SetupSignalHandler()

	providerResources := new(message.ProviderResources)

	snapshot := providerResources.GatewayAPIResources.Subscribe(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case s := <-snapshot:
				for key, value := range s.State {
					klog.Infof("key: %+v, value: %+v", key, value)
				}
				for _, update := range s.Updates {
					klog.Infof("[updates] key: %+v, value: %+v", update.Key, update.Value)
				}
			}
		}
	}()

	provider, err := NewProvider(ctx, providerResources)
	if err != nil {
		klog.Fatalf("failed to create provider: %v", err)
	}

	go func() {
		err = provider.Start(ctx) // block
		if err != nil {
			klog.Fatalf("failed to start provider: %v", err)
		}
	}()

	go func() {
		// 这里为了测试 list cache，所以这里先等待 cache 同步
		if ok := provider.mgr.GetCache().WaitForCacheSync(ctx); !ok {
			klog.Fatalf("failed to wait for caches to sync")
		}

		var gatewayClassList gatewayapiv1.GatewayClassList
		err = provider.mgr.GetCache().List(ctx, &gatewayClassList)
		if err != nil {
			klog.Fatalf("failed to list gatewayclass: %v", err)
		}

		for _, gatewayClass := range gatewayClassList.Items {
			// TODO: 这里 cache 还是全部的 gatewayclass 啊???
			klog.Infof("list from informer cache gatewayclass: %v", gatewayClass.Name)
		}

	}()

	klog.Info("started provider")
	<-ctx.Done()
}

func TestWatchable(test *testing.T) {
	ctx := ctrl.SetupSignalHandler()

	type TestWatchable struct {
		resources watchable.Map[string, string]
	}
	w := new(TestWatchable)
	snapshot := w.resources.Subscribe(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case s := <-snapshot:
				for key, value := range s.State {
					klog.Infof("key: %+v, value: %+v", key, value)
					// key: key1, value: value1
				}
				for _, update := range s.Updates {
					klog.Infof("[updates] key: %+v, value: %+v", update.Key, update.Value)
					// [updates] key: key1, value: value1
				}
			}
		}
	}()

	go func() {
		w.resources.Store("key1", "value1")
		w.resources.Store("key1", "value2")
		w.resources.Store("key1", "value3")
	}()

	klog.Info("started watchable")
	<-ctx.Done()
}
