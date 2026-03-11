package bootstrap

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/core/ingress/kube/common"
	"github.com/lx1036/gateway/pkg/core/ingress/translation"
	"github.com/lx1036/gateway/pkg/core/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pilot/pkg/serviceregistry/aggregate"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/config/mesh/meshwatcher"
	"k8s.io/client-go/tools/cache"
	"net"
	"time"

	"github.com/lx1036/gateway/pkg/core/kube"

	prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	configaggregate "istio.io/istio/pilot/pkg/config/aggregate"
	istiogrpc "istio.io/istio/pilot/pkg/grpc"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/server"
	kubecontroller "istio.io/istio/pilot/pkg/serviceregistry/kube/controller"
	"istio.io/istio/pilot/pkg/xds"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/keepalive"
	istiokube "istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/krt"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type RegistryOptions struct {
	// If FileDir is set, the below kubernetes options are ignored
	FileDir string

	Registries []string

	// Kubernetes controller options
	KubeOptions kubecontroller.Options
	// ClusterRegistriesNamespace specifies where the multi-cluster secret resides
	ClusterRegistriesNamespace string
	KubeConfig                 string

	// DistributionTracking control
	DistributionCacheRetention time.Duration

	// DistributionTracking control
	DistributionTrackingEnabled bool
}

type ServerArgs struct {
	GrpcKeepAliveOptions *keepalive.Options

	RegistryOptions RegistryOptions

	GrpcAddress string

	GatewaySelectorKey   string
	GatewaySelectorValue string
}

type Server struct {
	*ServerArgs

	server server.Instance

	kubeClient kube.Client

	environment      *model.Environment
	xdsServer        *xds.DiscoveryServer
	configController model.ConfigStoreController
	configStores     []model.ConfigStoreController

	grpcServer *grpc.Server
}

func NewServer(args *ServerArgs) (*Server, error) {
	e := model.NewEnvironment()
	e.DomainSuffix = constants.DefaultClusterLocalDomain
	//e.SetLedger(buildLedger(args.RegistryOptions))
	ac := aggregate.NewController(aggregate.Options{
		MeshHolder: e,
	})
	e.ServiceDiscovery = ac

	s := &Server{
		ServerArgs: args,

		environment: e,
		server:      server.New(),
	}

	s.environment.Watcher = meshwatcher.NewTestWatcher(&v1alpha1.MeshConfig{})
	s.environment.Init()

	if err := s.initKubeClient(); err != nil {
		return nil, err
	}
	if err := s.initXdsServer(); err != nil {
		return nil, err
	}
	if err := s.initConfigController(); err != nil {
		return nil, err
	}

	s.server.RunComponent("kube-client", func(stop <-chan struct{}) error {
		s.kubeClient.RunAndWait(stop)
		return nil
	})

	return s, nil
}

func (s *Server) Start(stop <-chan struct{}) error {
	if err := s.server.Start(stop); err != nil {
		return err
	}

	if !s.waitForCacheSync(stop) {
		return fmt.Errorf("failed to sync cache")
	}

	// Inform Discovery Server so that it can start accepting connections.
	s.xdsServer.CachesSynced()

	grpcListener, err := net.Listen("tcp", s.GrpcAddress)
	if err != nil {
		return err
	}
	go func() {
		klog.Infof("starting gRPC discovery service at %s", grpcListener.Addr())
		if err := s.grpcServer.Serve(grpcListener); err != nil {
			klog.Errorf("error serving GRPC server: %v", err)
		}
	}()

	return nil
}

func (s *Server) initKubeClient() error {
	if s.kubeClient != nil {
		// Already initialized by startup arguments
		return nil
	}

	klog.Infof("Initializing Kubernetes client")

	kubeRestConfig, err := istiokube.DefaultRestConfig(s.RegistryOptions.KubeConfig, "", func(config *rest.Config) {
		config.QPS = s.RegistryOptions.KubeOptions.KubernetesAPIQPS
		config.Burst = s.RegistryOptions.KubeOptions.KubernetesAPIBurst
	})
	if err != nil {
		return fmt.Errorf("failed creating kube config: %v", err)
	}
	s.kubeClient, err = kube.NewClient(istiokube.NewClientConfigForRestConfig(kubeRestConfig), "higress")
	if err != nil {
		return fmt.Errorf("failed creating kube client: %v", err)
	}
	s.kubeClient = kube.EnableCrdWatcher(s.kubeClient)

	return nil
}

func (s *Server) initXdsServer() error {
	klog.Infof("Initializing XDS server")

	s.xdsServer = xds.NewDiscoveryServer(s.environment, s.RegistryOptions.KubeOptions.ClusterAliases, krt.GlobalDebugHandler)
	// TODO: Generators

	for _, schema := range collections.Pilot.All() {
		gvkStr := schema.GroupVersionKind().String()
		klog.Infof("Initializing XDS server schema %s", gvkStr)
	}

	s.xdsServer.ProxyNeedsPush = func(proxy *model.Proxy, req *model.PushRequest) (*model.PushRequest, bool) {
		return req, true
	}

	s.server.RunComponent("xds-server", func(stop <-chan struct{}) error {
		klog.Infof("Starting ADS server")
		s.xdsServer.Start(stop)
		return nil
	})

	interceptors := []grpc.UnaryServerInterceptor{
		// setup server prometheus monitoring (as final interceptor in chain)
		prometheus.UnaryServerInterceptor,
	}
	grpcOptions := istiogrpc.ServerOptions(s.GrpcKeepAliveOptions, func(i int64) {}, interceptors...)
	s.grpcServer = grpc.NewServer(grpcOptions...)
	s.xdsServer.Register(s.grpcServer)
	reflection.Register(s.grpcServer)
	return nil
}

func (s *Server) waitForCacheSync(stop <-chan struct{}) bool {
	start := time.Now()
	klog.Info("Waiting for caches to be synced")
	if !cache.WaitForCacheSync(stop, s.cachesSynced) {
		klog.Errorf("Failed waiting for cache sync")
		return false
	}

	klog.Infof("All controller caches have been synced up in %v", time.Since(start))

	// 检查是否还有 config updates 还没有 processed
	expected := s.xdsServer.InboundUpdates.Load()
	if !cache.WaitForCacheSync(stop, func() bool {
		return s.pushContextReady(expected)
	}) {
		klog.Errorf("Failed waiting for push context initialization")
		return false
	}

	return true
}

// pushContextReady indicates whether push context has processed all inbound config updates.
func (s *Server) pushContextReady(expected int64) bool {
	committed := s.xdsServer.CommittedUpdates.Load()
	if committed < expected {
		klog.Warningf("Waiting for pushcontext to process inbound updates, inbound: %v, committed : %v", expected, committed)
		return false
	}
	return true
}

// cachesSynced checks whether caches have been synced.
func (s *Server) cachesSynced() bool {
	return s.configController.HasSynced()
}

func (s *Server) initConfigController() error {
	// INFO: 自定义逻辑, ingressConfig 是 configStore 之一，实现 xds://127.0.0.1:15051
	ns := util.PodNamespace
	options := common.Options{
		Enable:    true,
		ClusterId: s.RegistryOptions.KubeOptions.ClusterID,
		//IngressClass:         s.IngressClass,
		//WatchNamespace:       s.WatchNamespace,
		//EnableStatus:         s.EnableStatus,
		SystemNamespace: util.PodNamespace,
		//GatewaySelectorKey:   s.GatewaySelectorKey,
		//GatewaySelectorValue: s.GatewaySelectorValue,
		//GatewayHttpPort:      s.GatewayHttpPort,
		//GatewayHttpsPort:     s.GatewayHttpsPort,
	}
	ingressConfig := translation.NewIngressTranslation(s.kubeClient, s.xdsServer, ns, options)
	ingressConfig.AddLocalCluster(options)

	s.configStores = append(s.configStores, ingressConfig)

	// Wrap the config controller with a cache.
	aggregateConfigController, err := configaggregate.MakeCache(s.configStores)
	if err != nil {
		return err
	}
	s.configController = aggregateConfigController

	// Create the config store.
	s.environment.ConfigStore = aggregateConfigController

	// Defer starting the controller until after the service is created.
	s.server.RunComponent("config-controller", func(stop <-chan struct{}) error {
		go s.configController.Run(stop)
		return nil
	})

	return nil
}

// initRegistryEventHandlers sets up event handlers for config updates
func (s *Server) initRegistryEventHandlers() error {
	klog.Info("initializing registry event handlers")

	configHandler := func(prev config.Config, curr config.Config, event model.Event) {
		// For update events, trigger push only if spec has changed.
		pushReq := &model.PushRequest{
			Full: true,
			ConfigsUpdated: map[model.ConfigKey]struct{}{{
				Kind:      gvk.MustToKind(curr.GroupVersionKind),
				Name:      curr.Name,
				Namespace: curr.Namespace,
			}: {}},
			Reason: model.NewReasonStats(model.ConfigUpdate),
		}

		s.xdsServer.ConfigUpdate(pushReq)
	}

	schemas := common.IngressIR.All()
	for _, schema := range schemas {
		s.configController.RegisterEventHandler(schema.GroupVersionKind(), configHandler)
	}
	return nil
}

func (s *Server) WaitUntilCompletion() {
	s.server.Wait()
}
