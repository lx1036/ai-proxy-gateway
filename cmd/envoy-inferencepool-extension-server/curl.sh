

kubectl port-forward -n envoy-gateway-system svc/envoy-envoy-gateway-system-inference-pool-with-httprou-662a29ee 7788:7788


curl -i -H "Content-Type: application/json" \
  -d '{
        "model": "some-cool-self-hosted-model",
        "messages": [
            {
                "role": "system",
                "content": "Hi."
            }
        ]
    }' \
  localhost:7788/v1/chat/completions

HTTP/1.1 200 OK
x-went-into-resp-headers: true
content-type: application/json
testupstream-id: test
x-model: some-cool-self-hosted-model
date: Wed, 18 Mar 2026 11:54:50 GMT
transfer-encoding: chunked

{"choices":[{"finish_reason":"stop","index":0,"message":{"content":"You're gonna need a bigger boat.","role":"assistant"}}],"usage":{"completion_tokens":100,"prompt_tokens":1,"total_tokens":300}}%


kubectl port-forward -n envoy-gateway-system pod/envoy-envoy-gateway-system-inference-pool-with-httprou-662dts4j 19000:19000
curl -s "http://localhost:19000/config_dump" | yq -P > envpy-httproute-inferencepool.yaml
