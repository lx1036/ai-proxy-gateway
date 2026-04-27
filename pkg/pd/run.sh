
# 【vLLM+LMCache实战：单卡也能玩PD分离】https://mp.weixin.qq.com/s/ajuajelcUqghl1PiW49OfA


#--gpu-memory-utilization 0.3 - 单卡要跑两个 vLLM 实例，每个只能占 30% 显存
#--no-enable-prefix-caching - 关掉 vLLM 本地 prefix cache。不关的话，重复请求会被 GPU 显存里的 cache 吃掉，LMCache 根本不会被触发，看起来就像 LMCache 没起作用
#--enforce-eager - 关掉 CUDA Graph，避免内存问题
#kv_role - prefiller 是 kv_producer，decoder 是 kv_consumer
#lmcache_rpc_port - 两边一个叫 producer1 一个叫 consumer1，LMCache 内部靠这个配对


# 1. LMCache Server（共享 KV 池）
python3 -m lmcache.v1.server localhost 8500



# 2. prefill
UCX_TLS=cuda_ipc,cuda_copy,tcp \
    LMCACHE_CONFIG_FILE=lmcache-prefill-config.yaml \
    LMCACHE_USE_EXPERIMENTAL=True \
    VLLM_ENABLE_V1_MULTIPROCESSING=1 \
    VLLM_WORKER_MULTIPROC_METHOD=spawn \
    CUDA_VISIBLE_DEVICES=0 \
    vllm serve "$MODEL" \
    --port 8100 \
    --enforce-eager \
    --gpu-memory-utilization 0.3 \
    --no-enable-prefix-caching \
    --max-model-len 1000 \
    --max-num-seqs 16 \
    --kv-transfer-config \
    '{"kv_connector":"LMCacheConnectorV1","kv_role":"kv_producer","kv_connector_extra_config":{"discard_partial_chunks":false,"lmcache_rpc_port":"producer1"}}'


# 3. decode
UCX_TLS=cuda_ipc,cuda_copy,tcp \
    LMCACHE_CONFIG_FILE=lmcache-decode-config.yaml \
    LMCACHE_USE_EXPERIMENTAL=True \
    VLLM_ENABLE_V1_MULTIPROCESSING=1 \
    VLLM_WORKER_MULTIPROC_METHOD=spawn \
    CUDA_VISIBLE_DEVICES=0 \
    vllm serve "$MODEL" \
    --port 8200 \
    --enforce-eager \
    --gpu-memory-utilization 0.3 \
    --no-enable-prefix-caching \
    --max-model-len 1000 \
    --max-num-seqs 16 \
    --kv-transfer-config \
    '{"kv_connector":"LMCacheConnectorV1","kv_role":"kv_consumer","kv_connector_extra_config":{"discard_partial_chunks":false,"lmcache_rpc_port":"consumer1"}}'


# 4. router
cd ~/k8s/vllm/examples/others/lmcache/disagg_prefill_lmcache_v1
python3 disagg_proxy_server.py \
    --host localhost --port 9000 \
    --prefiller-host localhost --prefiller-port 8100 \
    --decoder-host localhost --decoder-port 8200


# 5. benchmark

cd ~/k8s/vllm/benchmarks
# 第一次（cold，KV 写入共享池）
vllm bench serve --port 9000 --seed 42 \
    --model meta-llama/Llama-3.2-1B-Instruct \
    --dataset-name random --random-input-len 768 --random-output-len 20 \
    --num-prompts 50 --burstiness 100 --request-rate 2
# 第二次（hot，应该命中共享池）
vllm bench serve --port 9000 --seed 42 \
    --model meta-llama/Llama-3.2-1B-Instruct \
    --dataset-name random --random-input-len 768 --random-output-len 20 \
    --num-prompts 50 --burstiness 100 --request-rate 2

#重点：random 数据集配合固定 --seed，两次生成的 prompt 内容完全相同，才能测出缓存命中效果。
#Input 长度选 768 不是随意的。LMCache 的 chunk_size=256，700 token 只能存 2 个完整 chunk (512 token)，命中率天花板 73%。改成 768（= 256×3）能接近 100% 命中。






