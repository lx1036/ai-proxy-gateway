


# 直接访问 vllm pod
k port-forward -n envoy-gateway-system pod/vllm-1p1d-lmcache-roleset-bzq69-decode-6f85db55df-0 18080:8000

curl http://localhost:18080/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
         "model": "qwen",
         "messages": [
             {"role": "system", "content": "You are a helpful assistant."},
             {"role": "user", "content": "你是谁，你知道中国首都是哪里"}
         ],
          "stream": false,
         "max_tokens": 500
     }'


# 访问 llmrouter 网关
curl -v http://localhost:8000/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
         "model": "qwen",
         "messages": [
             {"role": "system", "content": "You are a helpful assistant."},
             {"role": "user", "content": "你是谁，你知道中国首都是哪里"}
         ],
          "stream": false,
         "max_tokens": 500
     }'

