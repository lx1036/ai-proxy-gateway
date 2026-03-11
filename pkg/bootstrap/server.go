package bootstrap

import (
	"fmt"
	gatewayGrpc "github.com/lx1036/gateway/pkg/grpc"
	gatewayKeepalive "github.com/lx1036/gateway/pkg/keepalive"
	"github.com/lx1036/gateway/pkg/networking/core"
	"github.com/lx1036/gateway/pkg/xds"
	v3 "github.com/lx1036/gateway/pkg/xds/v3"
	"net"

	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/lx1036/gateway/pkg/kube"
	"github.com/lx1036/gateway/pkg/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type Server struct {
	xdsServer *xds.DiscoveryServer

	configController model.ConfigStoreController
	// TODO: 目前只实现 Kubernetes ConfigStore, 后续实现 XDS/File
	ConfigStores []model.ConfigStoreController

	environment *model.Environment
	kubeClient  kube.Client

	clusterID string

	grpcServer  *grpc.Server
	grpcAddress string
}

func NewServer(args *PilotArgs, initFuncs ...func(*Server)) (*Server, error) {
	/**
	1. xdsServer -> 实现 envoy xds 接口：StreamAggregatedResources()/DeltaAggregatedResources()
	2.
	ConfigGenerator ->
	*/

	e := model.NewEnvironment()
	//e.DomainSuffix = args.RegistryOptions.KubeOptions.DomainSuffix

	s := &Server{
		environment: e,
		clusterID:   "Kubernetes",

		grpcAddress: args.ServerOptions.GRPCAddr,
	}

	s.xdsServer = xds.NewDiscoveryServer(e)
	configGen := core.NewConfigGenerator(s.xdsServer.Cache)

	s.initServers(args)

	//s.initMeshConfiguration(args, s.fileWatcher)

	s.InitDiscoveryServerGenerators(configGen)

	if err := s.initControllers(args); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) Start(stop <-chan struct{}) error {

	// grpc server
	grpcListener, err := net.Listen("tcp", s.grpcAddress) // grpc://*:15010
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

// initServers initializes http and grpc servers
func (s *Server) initServers(args *PilotArgs) {
	s.initGrpcServer(args.KeepaliveOptions)
}

func (s *Server) initControllers(args *PilotArgs) error {
	klog.Info("initializing controllers")

	if err := s.initConfigController(args); err != nil {
		return fmt.Errorf("error initializing config controller: %v", err)
	}
	if err := s.initServiceController(args); err != nil {
		return fmt.Errorf("error initializing service controllers: %v", err)
	}

	return nil
}

func (s *Server) initGrpcServer(options *gatewayKeepalive.Options) {
	interceptors := []grpc.UnaryServerInterceptor{
		// setup server prometheus monitoring (as final interceptor in chain)
		grpcprom.UnaryServerInterceptor,
	}
	grpcOptions := gatewayGrpc.ServerOptions(options, interceptors...)
	s.grpcServer = grpc.NewServer(grpcOptions...)
	s.xdsServer.Register(s.grpcServer)
	reflection.Register(s.grpcServer)
}

func (s *Server) WaitUntilCompletion() {
	//s.server.Wait()
}

func (s *Server) initKubeClient(args *PilotArgs) error {
	if s.kubeClient != nil {
		// Already initialized by startup arguments
		return nil
	}

	kubeRestConfig, err := kube.DefaultRestConfig(args.RegistryOptions.KubeConfig, "", func(config *rest.Config) {
		//config.QPS = args.RegistryOptions.KubeOptions.KubernetesAPIQPS
		//config.Burst = args.RegistryOptions.KubeOptions.KubernetesAPIBurst
	})
	if err != nil {
		return fmt.Errorf("failed creating kube config: %v", err)
	}

	s.kubeClient, err = kube.NewClient(kube.NewClientConfigForRestConfig(kubeRestConfig), s.clusterID)
	if err != nil {
		return fmt.Errorf("failed creating kube client: %v", err)
	}
	s.kubeClient = kube.EnableCrdWatcher(s.kubeClient)

	return nil
}

func (s *Server) InitDiscoveryServerGenerators(cg *core.ConfigGenerator) {
	generators := map[string]xds.XdsResourceGenerator{}

	generators[v3.ClusterType] = &xds.CdsGenerator{ConfigGenerator: cg}
	//generators[v3.ListenerType] = &xds.LdsGenerator{ConfigGenerator: cg}
	//generators[v3.RouteType] = &xds.RdsGenerator{ConfigGenerator: cg}
	//generators[v3.EndpointType] = edsGen

	s.xdsServer.Generators = generators
}
