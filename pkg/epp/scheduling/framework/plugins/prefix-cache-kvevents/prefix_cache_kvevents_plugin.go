package prefix_cache_kvevents

/**

plugins:
  - type: precise-prefix-cache-scorer
    parameters:
        tokenProcessorConfig:
          blockSize: 16
          hashSeed: "12345"
        kvEventsConfig:
          topicFilter: "kv@"
          concurrency: 4
          discoverPods: true    # enables automatic pod discovery for active-active HA
          podDiscoveryConfig:
            socketPort: 5556
        indexerConfig:
          prefixStoreConfig:
            cacheSize: 500000
            blockSize: 256
          kvBlockIndexConfig:
            inMemoryConfig:
              size: 100000000
              podCacheSize: 10
            enableMetrics: true
          tokenizersPoolConfig:
            modelName: hf-repo/model-name
            workersCount: 8
            hf:
              huggingFaceToken: your_hf_token_here    # automatically set by `HF_TOKEN` environment variable
              tokenizersCacheDir: /tmp/tokenizers

*/

type PrecisePrefixCacheScorer struct {
}
