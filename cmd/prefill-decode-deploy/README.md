
# 目标
既可以单机/多机部署，也支持 PD 分离部署。

# 单机部署
Deployment/LWS(WorkerTemplate)

# 多机部署
* 1、Docker 容器方式实现多机部署
* 2、LWS(测试无法单机部署)，必须架构：Leader + Worker.
  * 只有 Leader 接收 Router 转发的流量。


# PD 分离部署
StormService：xPyD 架构
* x Prefill 节点可以接收流量，和 Leader+Worker 多机架构还不一样。
