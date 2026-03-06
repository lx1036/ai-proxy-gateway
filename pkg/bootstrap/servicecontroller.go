package bootstrap

/**
https://www.zhaohuabing.com/post/2019-10-21-pilot-discovery-code-analysis/

Service Controller用于管理各种Service Registry，提出服务发现数据，目前Istio支持的Service Registry包括：

Kubernetes：对接Kubernetes Registry，可以将Kubernetes中定义的Service和Instance采集到Istio中。
Consul： 对接Consul Catalog，将Consul中定义的Service采集到Istio中。
MCP： 和MCP config controller类似，从MCP Server中获取Service和Service Instance。
Memory： 一个内存中的Service Controller实现，主要用于测试。
*/

func (s *Server) initServiceControllers(args *PilotArgs) error {

}
