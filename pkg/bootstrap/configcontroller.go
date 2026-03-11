package bootstrap

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/config/kube/crdclient"
	"github.com/lx1036/gateway/pkg/config/schema/collections"
	"k8s.io/klog/v2"
	"net/url"
)

/**
https://www.zhaohuabing.com/post/2019-10-21-pilot-discovery-code-analysis/

Config Controller用于管理各种配置数据，包括用户创建的流量管理规则和策略。Istio目前支持三种类型的Config Controller：

Kubernetes：使用Kubernetes来作为配置数据的存储，该方式的直接依附于Kubernetes强大的CRD机制来存储配置数据，简单方便，是Istio最开始使用的配置存储方案。
MCP (Mesh Configuration Protocol)：使用Kubernetes来存储配置数据导致了Istio和Kubernetes的耦合，限制了Istio在非Kubernetes环境下的运用。为了解决该耦合，Istio社区提出了MCP，MCP定义了一个向Istio控制面下发配置数据的标准协议，Istio Pilot作为MCP Client，任何实现了MCP协议的Server都可以通过MCP协议向Pilot下发配置，从而解除了Istio和Kubernetes的耦合。如果想要了解更多关于MCP的内容，请参考文后的链接。
Memory：一个在内存中的Config Controller实现，主要用于测试。
目前Istio的配置包括：

Virtual Service: 定义流量路由规则。
Destination Rule: 定义和一个服务或者subset相关的流量处理规则，包括负载均衡策略，连接池大小，断路器设置，subset定义等等。
Gateway: 定义入口网关上对外暴露的服务。
Service Entry: 通过定义一个Service Entry可以将一个外部服务手动添加到服务网格中。
Envoy Filter: 通过Pilot在Envoy的配置中添加一个自定义的Filter。

*/

type ConfigSourceAddressScheme string

const (
	// XDS e.g. xds://127.0.0.1:15051
	XDS ConfigSourceAddressScheme = "xds"

	// Kubernetes e.g. k8s://
	Kubernetes ConfigSourceAddressScheme = "k8s"
)

func (s *Server) initConfigController(args *PilotArgs) error {
	stop := make(chan struct{})

	if err := s.initConfigSources(args); err != nil {
		return err
	}

	for _, store := range s.ConfigStores {
		go store.Run(stop)
	}

	return nil
}

// TODO: 目前只实现 Kubernetes ConfigStore, 后续实现 XDS/File
func (s *Server) initConfigSources(args *PilotArgs) (err error) {
	err2 := s.initK8SConfigStore(args)
	if err2 != nil {
		klog.Warningf("Error loading k8s: %v", err2)
		return err2
	}

	return nil
}

// K8S ConfigStore 主要使用 CrdClient 从 k8s 里获取 resource
func (s *Server) initK8SConfigStore(args *PilotArgs) error {
	if s.kubeClient == nil {
		return nil
	}
	configController := s.makeKubeConfigController(args)
	s.ConfigStores = append(s.ConfigStores, configController)

	// gateway
	//gwc := gateway.NewController(s.kubeClient, s.kubeClient.CrdWatcher().WaitForCRD, args.RegistryOptions.KubeOptions, s.xdsServer)

}

func (s *Server) makeKubeConfigController(args *PilotArgs) *crdclient.Client {
	// gateway-api/gateway-api-inference-extension CR, skip Ingress
	schemas := collections.GatewayAPI()
	schemas = schemas.Add(collections.InferencePool)

	opts := crdclient.Option{
		Revision:     args.Revision,
		DomainSuffix: args.RegistryOptions.KubeOptions.DomainSuffix,
		Identifier:   "crd-controller",
		KrtDebugger:  args.KrtDebugger,
	}

	return crdclient.NewForSchemas(s.kubeClient, opts, schemas)
}
