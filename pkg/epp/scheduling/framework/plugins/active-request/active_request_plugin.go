package active_request

// https://github.com/llm-d/llm-d-inference-scheduler/pull/297

/**
该插件可以解决高并发场景下，scheduler 会瞬时把大量请求转给同一个 endpoint 的问题。可以参见 issue:
https://github.com/kubernetes-sigs/gateway-api-inference-extension/issues/1700

同时，也解决了这个疑问：https://github.com/llm-d/llm-d-inference-scheduler/issues/228

active_request_plugin 解决方式：
1、获取 endpoint 的 active request 数
2、scheduler 会根据每一个 endpoint 的 active request 数量来 score，决定哪些 endpoint 是否 idle（超过阈值则为 busy），从而决定转发策略。
3、可以解决高并发场景下，scheduler 可能会瞬时把大量请求转给同一个 endpoint 的问题。
*/
