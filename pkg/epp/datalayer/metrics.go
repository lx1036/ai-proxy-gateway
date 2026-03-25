package datalayer

import "time"

// Metrics holds the latest metrics snapshot scraped from a pod.
type Metrics struct {
	// ActiveModels is a set of models(including LoRA adapters) that are currently cached to GPU.
	ActiveModels  map[string]int
	WaitingModels map[string]int
	// MaxActiveModels is the maximum number of models that can be loaded to GPU.
	MaxActiveModels         int
	RunningRequestsSize     int
	WaitingQueueSize        int
	KVCacheUsagePercent     float64
	KvCacheMaxTokenCapacity int
	CacheBlockSize          int
	// Number of GPU blocks in the model server for KV Cache.
	CacheNumGPUBlocks int

	// UpdateTime records the last time when the metrics were updated.
	UpdateTime time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{
		ActiveModels:  make(map[string]int),
		WaitingModels: make(map[string]int),
	}
}

func (m *Metrics) Clone() *Metrics {
	if m == nil {
		return nil
	}

	activeModels := make(map[string]int, len(m.ActiveModels))
	for key, value := range m.ActiveModels {
		activeModels[key] = value
	}
	waitingModels := make(map[string]int, len(m.WaitingModels))
	for key, value := range m.WaitingModels {
		waitingModels[key] = value
	}

	return &Metrics{
		ActiveModels:            activeModels,
		WaitingModels:           waitingModels,
		MaxActiveModels:         m.MaxActiveModels,
		RunningRequestsSize:     m.RunningRequestsSize,
		WaitingQueueSize:        m.WaitingQueueSize,
		KVCacheUsagePercent:     m.KVCacheUsagePercent,
		KvCacheMaxTokenCapacity: m.KvCacheMaxTokenCapacity,
		CacheBlockSize:          m.CacheBlockSize,
		CacheNumGPUBlocks:       m.CacheNumGPUBlocks,
		UpdateTime:              m.UpdateTime,
	}
}
