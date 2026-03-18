


# 1. 目标
envoy-gateway + HTTPRoute + InferencePool + vllm-simulator-backend

这需要 envoy-gateway ExtensionServer 机制，具体做了什么？

https://aigateway.envoyproxy.io/docs/0.3/capabilities/inference/httproute-inferencepool

```shell

kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v0.5.1/manifests.yaml

kubectl apply -f https://raw.githubusercontent.com/envoyproxy/ai-gateway/release/v0.3/examples/inference-pool/config.yaml

kubectl rollout restart -n envoy-gateway-system deployment/envoy-gateway

kubectl wait --timeout=2m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available


kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/raw/v0.5.1/config/manifests/inferencemodel.yaml

kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/raw/v0.5.1/config/manifests/inferencepool-resources.yaml
```


# 2. 问题
EnvoyExtensionPolicy 是否可以替代掉 HTTPRoute + InferencePool + ExtensionServer? 
或者说，有了 EnvoyExtensionPolicy，为啥还有 ExtensionServer?
EnvoyExtensionPolicy + EnvoyPatchPolicy 是不是可以替代 ExtensionServer?
