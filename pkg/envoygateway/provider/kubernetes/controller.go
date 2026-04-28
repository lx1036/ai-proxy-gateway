package kubernetes

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/lx1036/gateway/pkg/envoygateway/message"
	"github.com/lx1036/gateway/pkg/envoygateway/scheme"

	envoygatewayv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	GatewayClassController = "gateway.envoyproxy.io/gatewayclass-controller"
)

type GatewayAPIReconciler struct {
	classController gatewayapiv1.GatewayController

	mgr             manager.Manager
	discoveryClient *discovery.DiscoveryClient

	extBackendGVKs []schema.GroupVersionKind

	resources *message.ProviderResources

	envoyExtensionPolicyCRDExists bool
	envoyPatchPolicyCRDExists     bool
}

func NewGatewayAPIReconciler(ctx context.Context, mgr manager.Manager, resources *message.ProviderResources) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("create discovery client: %w", err)
	}
	gatewayAPIReconciler := &GatewayAPIReconciler{
		classController: gatewayapiv1.GatewayController(GatewayClassController),

		mgr:             mgr,
		discoveryClient: discoveryClient,

		resources: resources,
	}

	// controller-runtime doesn't allow run controller with same name for more than once
	// see https://github.com/kubernetes-sigs/controller-runtime/blob/2b941650bce159006c88bd3ca0d132c7bc40e947/pkg/controller/name.go#L29
	name := fmt.Sprintf("gatewayapi-%d", time.Now().Unix())
	c, err := controller.New(name, mgr, controller.Options{
		Reconciler: gatewayAPIReconciler,
	})

	err = gatewayAPIReconciler.WatchResources(ctx, mgr, c)
	if err != nil {
		return fmt.Errorf("failed to watch resource error: %v", err)
	}

	//klog.Infof("new gateway api reconciler")

	return nil
}

func (gatewayAPIReconciler *GatewayAPIReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	/**
	INFO: 0. 构建 gateway-api 全局资源视图，每一个资源的更新（包括任意子资源），都会触发构建全局资源视图。
	*/

	// INFO: 1. 聚合所有 gateway-api resources. 一个 Reconcile update 多个 resources.
	gatewayClasses, err := gatewayAPIReconciler.ListGatewayClass(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list gatewayclass error: %w", err)
	}
	//klog.Infof("list gateway class %d", len(gatewayClasses))
	if len(gatewayClasses) == 0 {
		klog.Infof("no accepted gatewayclass")
		return reconcile.Result{}, nil
	}

	gatewayClassResources := make(message.GatewayAPIResources, 0, len(gatewayClasses))
	for _, gatewayClass := range gatewayClasses {
		gatewayAPIResource := message.NewGatewayAPIResource()
		gatewayAPIResource.GatewayClass = gatewayClass

		// 1. Add all gateways/routes to the gatewayAPIResource
		err = gatewayAPIReconciler.processGateways(ctx, gatewayClass, gatewayAPIResource)

		if gatewayAPIReconciler.envoyExtensionPolicyCRDExists {

			gatewayAPIReconciler.processEnvoyExtensionPolicy(ctx)

		}

		if gatewayAPIReconciler.envoyPatchPolicyCRDExists {

			err = gatewayAPIReconciler.processEnvoyPatchPolicy()
		}

		gatewayClassResources = append(gatewayClassResources, gatewayAPIResource)
	}

	// sort before store 为了：1. 避免重复的 resources 被 watchable layer 去更新 envoy xds 2. gateway-api layer 确保 resources 资源顺序
	//gatewayClassResources.Sort()

	// INFO: 2. Publish GatewayAPIResource
	//klog.Infof("list gatewayClassResources %d", len(gatewayClassResources))
	gatewayAPIReconciler.resources.GatewayAPIResources.Store(string(gatewayAPIReconciler.classController), &gatewayClassResources)

	klog.Infof("gateway-api reconcile end successfully...")
	return reconcile.Result{}, nil
}

func (gatewayAPIReconciler *GatewayAPIReconciler) WatchResources(ctx context.Context, mgr manager.Manager, c controller.Controller) error {
	// INFO: 1. Watch GatewayClass CR 必须安装
	if ok, err := gatewayAPIReconciler.CheckCRDExists(scheme.KindGatewayClass, gatewayapiv1.GroupVersion.String()); err != nil || !ok {
		klog.Fatalf("failed to check gatewayclass crd error: %v", err)
	}
	err := c.Watch(source.Kind(
		mgr.GetCache(),
		&gatewayapiv1.GatewayClass{},
		handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, object *gatewayapiv1.GatewayClass) []reconcile.Request {
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name: string(gatewayAPIReconciler.classController),
					},
				},
			}
		}),
		&predicate.TypedGenerationChangedPredicate[*gatewayapiv1.GatewayClass]{},
		predicate.NewTypedPredicateFuncs(func(gc *gatewayapiv1.GatewayClass) bool {
			// TODO: 这里 cache 还是全部的 gatewayclass 啊???
			return gc.Spec.ControllerName == gatewayAPIReconciler.classController
		}),
	))
	if err != nil {
		return fmt.Errorf("failed to watch %s: %v", scheme.KindGatewayClass, err)
	}

	// INFO: 2. Watch EnvoyProxy CR 可选
	ok, err := gatewayAPIReconciler.CheckCRDExists(scheme.KindEnvoyProxy, envoygatewayv1alpha1.GroupVersion.String())
	if err != nil {
		klog.Fatalf("failed to check gatewayclass crd error: %v", err)
	}
	if !ok {
		klog.Warningf("EnvoyProxy CRD not found, skipping EnvoyProxy watch")
	} else {
		err = c.Watch(source.Kind(
			mgr.GetCache(),
			&envoygatewayv1alpha1.EnvoyProxy{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, object *envoygatewayv1alpha1.EnvoyProxy) []reconcile.Request {
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: string(gatewayAPIReconciler.classController),
						},
					},
				}
			}),
			&predicate.TypedGenerationChangedPredicate[*envoygatewayv1alpha1.EnvoyProxy]{},
		))
		if err != nil {
			return fmt.Errorf("failed to watch %s: %v", scheme.KindEnvoyProxy, err)
		}
	}

	// INFO: 3. Watch Gateway CR
	ok, err = gatewayAPIReconciler.CheckCRDExists(scheme.KindGateway, gatewayapiv1.GroupVersion.String())
	if err != nil || !ok {
		klog.Fatalf("failed to check gateway crd error: %v", err)
	}
	err = c.Watch(source.Kind(
		mgr.GetCache(),
		&gatewayapiv1.Gateway{},
		handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, object *gatewayapiv1.Gateway) []reconcile.Request {
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name: string(gatewayAPIReconciler.classController),
					},
				},
			}
		}),
		&predicate.TypedGenerationChangedPredicate[*gatewayapiv1.Gateway]{},
		predicate.NewTypedPredicateFuncs(func(gateway *gatewayapiv1.Gateway) bool {
			var gatewayClass gatewayapiv1.GatewayClass
			if err := gatewayAPIReconciler.mgr.GetClient().Get(ctx, types.NamespacedName{Name: string(gateway.Spec.GatewayClassName)}, &gatewayClass); err != nil {
				klog.Errorf("failed to get gatewayclass %s: %v", gateway.Spec.GatewayClassName, err)
				return false
			}

			// 只会 watch gatewayclass GatewayClassController 所属的 gateway
			return gatewayClass.Spec.ControllerName == gatewayAPIReconciler.classController
		}),
	))
	if err != nil {
		return fmt.Errorf("failed to watch %s: %v", scheme.KindGateway, err)
	}
	err = mgr.GetFieldIndexer().IndexField(context.Background(), &gatewayapiv1.Gateway{}, "spec.gatewayClassName", func(object client.Object) []string {
		gateway := object.(*gatewayapiv1.Gateway)
		return []string{string(gateway.Spec.GatewayClassName)}
	})
	if err != nil {
		return fmt.Errorf("failed to index gateway: %v", err)
	}

	// INFO: 4. Watch HTTPRoute CR
	ok, err = gatewayAPIReconciler.CheckCRDExists(scheme.KindHTTPRoute, gatewayapiv1.GroupVersion.String())
	if err != nil || !ok {
		klog.Fatalf("failed to check HTTPRoute crd error: %v", err)
	}
	err = c.Watch(source.Kind(
		mgr.GetCache(),
		&gatewayapiv1.HTTPRoute{},
		handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, object *gatewayapiv1.HTTPRoute) []reconcile.Request {
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name: string(gatewayAPIReconciler.classController),
					},
				},
			}
		}),
		predicate.TypedGenerationChangedPredicate[*gatewayapiv1.HTTPRoute]{},
		predicate.TypedLabelChangedPredicate[*gatewayapiv1.HTTPRoute]{},
		predicate.TypedAnnotationChangedPredicate[*gatewayapiv1.HTTPRoute]{},
	))
	if err != nil {
		return fmt.Errorf("failed to watch %s: %v", scheme.KindHTTPRoute, err)
	}
	err = addHTTPRouteIndexers(ctx, mgr)
	if err != nil {
		return fmt.Errorf("failed to add Indexer for %s: %v", scheme.KindHTTPRoute, err)
	}

	// INFO:  Watch EnvoyExtensionPolicy CR
	gatewayAPIReconciler.envoyExtensionPolicyCRDExists, err = gatewayAPIReconciler.CheckCRDExists(envoygatewayv1alpha1.KindEnvoyExtensionPolicy, gatewayapiv1.GroupVersion.String())
	if err != nil {
		klog.Fatalf("failed to check %s crd error: %v", envoygatewayv1alpha1.KindEnvoyExtensionPolicy, err)
	}
	if !gatewayAPIReconciler.envoyExtensionPolicyCRDExists {
		klog.Infof("EnvoyExtensionPolicy CRD not found, skipping EnvoyExtensionPolicy watch")
	} else {
		err = c.Watch(source.Kind(
			mgr.GetCache(),
			&envoygatewayv1alpha1.EnvoyExtensionPolicy{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, object *envoygatewayv1alpha1.EnvoyExtensionPolicy) []reconcile.Request {
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: string(gatewayAPIReconciler.classController),
						},
					},
				}
			}),
			predicate.TypedGenerationChangedPredicate[*envoygatewayv1alpha1.EnvoyExtensionPolicy]{},
		))
		if err != nil {
			return fmt.Errorf("failed to watch %s: %v", envoygatewayv1alpha1.KindEnvoyExtensionPolicy, err)
		}

		err = addEnvoyExtensionPolicyIndexers(ctx, mgr)
		if err != nil {
			return fmt.Errorf("failed to add Indexer for %s: %v", envoygatewayv1alpha1.KindEnvoyExtensionPolicy, err)
		}
	}

	// INFO:  Watch EnvoyPatchPolicy CR
	gatewayAPIReconciler.envoyPatchPolicyCRDExists, err = gatewayAPIReconciler.CheckCRDExists(envoygatewayv1alpha1.KindEnvoyPatchPolicy, gatewayapiv1.GroupVersion.String())
	if err != nil {
		klog.Fatalf("failed to check HTTPRoute crd error: %v", err)
	}
	if !gatewayAPIReconciler.envoyPatchPolicyCRDExists {
		klog.Infof("EnvoyPatchPolicy CRD not found, skipping EnvoyPatchPolicy watch")
	} else {
		err = c.Watch(source.Kind(
			mgr.GetCache(),
			&envoygatewayv1alpha1.EnvoyPatchPolicy{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, object *envoygatewayv1alpha1.EnvoyPatchPolicy) []reconcile.Request {
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: string(gatewayAPIReconciler.classController),
						},
					},
				}
			}),
			predicate.TypedGenerationChangedPredicate[*envoygatewayv1alpha1.EnvoyPatchPolicy]{},
		))
		if err != nil {
			return fmt.Errorf("failed to watch %s: %v", envoygatewayv1alpha1.KindEnvoyPatchPolicy, err)
		}
	}

	// INFO: 5. Watch Custom GVK CR
	for _, gvk := range gatewayAPIReconciler.extBackendGVKs {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(gvk)
		err := c.Watch(source.Kind(
			mgr.GetCache(),
			u,
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, object *unstructured.Unstructured) []reconcile.Request {
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: string(gatewayAPIReconciler.classController),
						},
					},
				}
			}),
			predicate.TypedGenerationChangedPredicate[*unstructured.Unstructured]{},
		))
		if err != nil {
			return fmt.Errorf("failed to watch %s: %v", gvk.Kind, err)
		}

		klog.Infof("watch custom GVK %s backend resource", gvk.String())
	}

	return nil
}

func (gatewayAPIReconciler *GatewayAPIReconciler) CheckCRDExists(kind, groupVersion string) (bool, error) {
	exists := false
	err := retry.OnError(wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Second,
	}, func(err error) bool {
		return err != nil
	}, func() error {
		apiResourceList, err := gatewayAPIReconciler.discoveryClient.ServerResourcesForGroupVersion(groupVersion)
		if err != nil {
			// the CRD doesn't exist when the GroupVersion doesn't exist
			if errors.IsNotFound(err) || meta.IsNoMatchError(err) {
				return nil
			}

			return err
		}

		for _, resource := range apiResourceList.APIResources {
			if resource.Kind == kind {
				exists = true
			}
		}

		return nil
	})
	if err != nil {
		return false, fmt.Errorf("discover resources for %s: %w", groupVersion, err)
	}

	return exists, nil
}

func (gatewayAPIReconciler *GatewayAPIReconciler) ListGatewayClass(ctx context.Context) ([]*gatewayapiv1.GatewayClass, error) {
	var gatewayClassList gatewayapiv1.GatewayClassList
	if err := gatewayAPIReconciler.mgr.GetClient().List(ctx, &gatewayClassList); err != nil {
		return nil, fmt.Errorf("list gatewayclass error: %w", err)
	}

	var gatewayClasses []*gatewayapiv1.GatewayClass
	for key := range gatewayClassList.Items {
		item := gatewayClassList.Items[key]
		// INFO: 这里 list gatewayclass 还是全部的，没有考虑 Watch 里的过滤条件 gc.Spec.ControllerName == gatewayAPIReconciler.classController
		//  解释应该是：watch 应该是考虑了过滤条件，而这里的 GetClient().List() 应该是直接读 apiserver
		//klog.Infof("list gatewayclass %s", item.Name)
		if item.Spec.ControllerName == gatewayAPIReconciler.classController {
			// gateway class was marked for deletion, and finalizer are removed
			if !item.DeletionTimestamp.IsZero() && !slices.Contains(item.Finalizers, gatewayapiv1.GatewayClassFinalizerGatewaysExist) {
				continue
			}

			gatewayClasses = append(gatewayClasses, &item)
		}
	}

	return gatewayClasses, nil
}
