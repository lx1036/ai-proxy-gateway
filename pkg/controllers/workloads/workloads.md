


# 功能设计
LWS 设计文档（非常优秀）：https://docs.google.com/document/d/1C0wgkOdDov8fEsBNZF3wPwYv1njRuWBs2-BueymXyfM/edit?tab=t.0

* (1) 支持单机部署，替换掉目前的 sts/deployment 来单机部署。
* (2) 支持多副本的多机部署，支持多组 LLM 实例，每组 LLM 实例由多机组成。
* (3) 支持灰度发布 Rollout 和 Rolling update，即使用 sts partition RollingUpdate 功能。
  * 滚动更新也要按照 LLMInstance 实例来：每一组 group pods 滚动更新时，只有当前组实例(leader pod + worker pods)滚动更新完成，才会进行下一个实例滚动更新，避免 leader pods 依次去更新，但是 worker pods 还在慢慢更新。
* (4) 支持 gang scheduling。每组 LLM 实例需要支持 gang scheduling。
* (5) Topology-aware placement：支持 Topology-aware placement，这对一体机背靠背部署非常重要，可以节省一套交换机的钱。
  * 一体机有场景：4台机器，但是 node-0 和 node-1、node-2 和 node-3 之间是互联的，省一套交换机的钱。
* (6) failure handling：支持 All-or-nothing restart for failure handling，即一组实例全部重启，或者全部不重启。
* (7) 实现 PD 分离部署。LWS 可以支持。RBG 也可以支持。
  * SGLang LLM 推理框架基于 LWS 做的 PD 分离部署：https://docs.sglang.io/references/multi_node_deployment/lws_pd/lws_pd_deploy.html
  * SGLang LLM 推理框架基于 RBG 做的 PD 分离部署：https://docs.sglang.io/references/multi_node_deployment/rbg_pd/deepseekv32_pd.html


CR:
* LLMInstance
* LLMInstanceSet

