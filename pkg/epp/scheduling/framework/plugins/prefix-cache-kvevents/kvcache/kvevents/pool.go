package kvevents

import (
	"context"
	"k8s.io/klog/v2"
	"strings"
	"sync"

	"k8s.io/client-go/util/workqueue"
)

const (
	defaultPodSelector           = "networking.lx1036.ai/inferenceServing=true"
	defaultEventSourceDeviceTier = "GPU"
)

type Config struct {
	// ZMQEndpoint is the ZMQ address to connect to (e.g., "tcp://indexer:5557").
	ZMQEndpoint string `json:"zmqEndpoint,omitempty"`
	// TopicFilter is the ZMQ subscription filter (e.g., "kv@").
	TopicFilter string `json:"topicFilter"`
	// Concurrency is the number of parallel workers to run.
	Concurrency int `json:"concurrency"`
	// EngineType selects the inference engine adapter ("vllm" or "sglang").
	// Default: "vllm".
	EngineType string `json:"engineType,omitempty"`
	// DiscoverPods enables the Kubernetes pod reconciler for automatic
	// per-pod subscriber management. When enabled, the reconciler watches
	// Kubernetes pods and creates/removes ZMQ subscribers dynamically.
	DiscoverPods bool `json:"discoverPods"`
	// PodDiscoveryConfig holds the configuration for pod discovery.
	// Only used when DiscoverPods is true.
	PodDiscoveryConfig *PodDiscoveryConfig `json:"podDiscoveryConfig,omitempty"`
}

// PodDiscoveryConfig holds configuration for the Kubernetes pod reconciler.
type PodDiscoveryConfig struct {
	// PodLabelSelector is a label selector string for filtering which pods to watch.
	// Example: "app=vllm" or "app=vllm,tier=gpu"
	PodLabelSelector string `json:"podLabelSelector"`
	// PodNamespace limits the reconciler to watch pods in a specific namespace.
	// If empty, watches all namespaces (requires appropriate RBAC).
	PodNamespace string `json:"podNamespace,omitempty"`
	// SocketPort is the port number where vLLM pods expose their ZMQ socket.
	// The reconciler will connect to tcp://<PodIP>:<SocketPort>
	// Default: 5557
	SocketPort int `json:"socketPort"`
}

func DefaultConfig() *Config {
	return &Config{
		TopicFilter:        "kv@",
		Concurrency:        4,
		DiscoverPods:       true,
		PodDiscoveryConfig: DefaultPodReconcilerConfig(),
	}
}

func DefaultPodReconcilerConfig() *PodDiscoveryConfig {
	return &PodDiscoveryConfig{
		PodLabelSelector: defaultPodSelector,
		SocketPort:       5557,
	}
}

type EngineAdapter interface {
	// ParseMessage parses a raw transport message into domain data.
	// It extracts pod identity and model name from the topic,
	// and decodes the payload into an EventBatch.
	ParseMessage(msg *RawMessage) (podID, modelName string, batch EventBatch, err error)
}

type Pool struct {
	wg *sync.WaitGroup

	queues      []workqueue.TypedRateLimitingInterface[*RawMessage]
	concurrency int // can replace use with len(queues)

	adapter EngineAdapter
}

func NewPool(cfg *Config, adapter EngineAdapter) *Pool {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	pool := &Pool{
		wg:          &sync.WaitGroup{},
		queues:      make([]workqueue.TypedRateLimitingInterface[*RawMessage], cfg.Concurrency),
		concurrency: cfg.Concurrency,
	}

	for i := 0; i < pool.concurrency; i++ {
		pool.queues[i] = workqueue.NewTypedRateLimitingQueue[*RawMessage](workqueue.DefaultTypedControllerRateLimiter[*RawMessage]())
	}

	return pool
}

func (pool *Pool) Start(ctx context.Context) {
	pool.wg.Add(pool.concurrency)
	for i := 0; i < pool.concurrency; i++ {
		go pool.worker(ctx, i)
	}
}

func (pool *Pool) Shutdown() {
	for _, queue := range pool.queues {
		queue.ShutDown()
	}

	pool.wg.Wait()
}

func (pool *Pool) worker(ctx context.Context, workerIndex int) {
	defer pool.wg.Done()
	queue := pool.queues[workerIndex]
	for {
		task, shutdown := queue.Get()
		if shutdown {
			return
		}

		if err := pool.processTask(ctx, task); err != nil {
			queue.AddRateLimited(task)
			continue
		}
		// remove the task from the queue.
		queue.Forget(task)
		queue.Done(task)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// EventType represents the type of KV-cache event.
type EventType string

const (
	// EventTypeBlockStored indicates blocks were added to cache.
	EventTypeBlockStored EventType = "BlockStored"
	// EventTypeBlockRemoved indicates blocks were evicted from cache.
	EventTypeBlockRemoved EventType = "BlockRemoved"
	// EventTypeAllBlocksCleared indicates entire cache was cleared.
	EventTypeAllBlocksCleared EventType = "AllBlocksCleared"
)

// GenericEvent represents a KV-cache event containing already-parsed data.
type GenericEvent interface {
	// Type returns the event type.
	Type() EventType
}

// EventBatch represents a batch of generic events from an inference engine.
type EventBatch struct {
	Timestamp float64
	Events    []GenericEvent
}

/**

/Users/lx1036/Code/k8s/vllm/vllm/distributed/kv_events.py

MEDIUM_GPU = "GPU"

class BlockStored(KVCacheEvent):
    block_hashes: list[ExternalBlockHash]
    parent_block_hash: Optional[ExternalBlockHash]
    token_ids: list[int]
    block_size: int
    lora_id: Optional[int]
    medium: Optional[str]


class BlockRemoved(KVCacheEvent):
    block_hashes: list[ExternalBlockHash]
    medium: Optional[str]


class AllBlocksCleared(KVCacheEvent):
    pass
*/

// BlockStoredEvent represents blocks being added to the cache.
type BlockStoredEvent struct {
	BlockHashes []uint64
	Tokens      []uint32
	ParentHash  uint64
	DeviceTier  string
	LoraID      *int
	LoraName    *string
	ExtraKeys   [][]any
}

func (*BlockStoredEvent) Type() EventType {
	return EventTypeBlockStored
}

func (pool *Pool) processTask(ctx context.Context, task *RawMessage) error {
	podID, modelName, batch, err := pool.adapter.ParseMessage(task)
	if err != nil {
		klog.Errorf("failed to parse message: %v", err)
		return err
	}

	pool.processEventBatch(ctx, &batch, podID, modelName)
}

func (pool *Pool) processEventBatch(ctx context.Context, batch *EventBatch, podIdentifier, modelName string) {
	for _, genericEvent := range batch.Events {
		switch genericEvent.Type() {
		case EventTypeBlockStored:
			klog.Infof("block stored: %s", event.BlockID)

			event := genericEvent.(*BlockStoredEvent)
			// Default to gpu.
			deviceTier := defaultEventSourceDeviceTier
			if event.DeviceTier != "" {
				deviceTier = strings.ToLower(event.DeviceTier)
			}

			// Use LoRA name as model identifier if available, otherwise fall back to base model name.
			effectiveModelName := modelName
			if ev.LoraName != nil && *ev.LoraName != "" {
				effectiveModelName = *ev.LoraName
			}

			requestKeys, err := p.tokenProcessor.TokensToKVBlockKeys(parentRequestKey, ev.Tokens, effectiveModelName, extraFeatures)
			if err != nil {
				klog.Errorf("failed to convert tokens to block keys: %v", err)
				continue
			}

			// Only proceed if we have valid keys to add.
			if len(engineKeys) > 0 {
				if err := p.index.Add(ctx, engineKeys, requestKeys, podEntries); err != nil {

				}

			}



		case EventTypeBlockRemoved:
			klog.Infof("block removed: %s", event.BlockID)
		case EventTypeAllBlocksCleared:

		default:
			klog.Infof("unknown event type: %s", genericEvent.Type())
		}
	}

}

func (pool *Pool) AddTask(task *RawMessage) {
	queue := pool.queues[0]

}
