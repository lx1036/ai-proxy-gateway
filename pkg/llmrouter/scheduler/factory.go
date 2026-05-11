package scheduler

import (
	"github.com/lx1036/gateway/pkg/llmrouter/scheduler/plugins"
	"k8s.io/apimachinery/pkg/runtime"
)

type ScorePluginBuilder = func(arg runtime.RawExtension) ScorePlugin
type FilterPluginBuilder = func(arg runtime.RawExtension) FilterPlugin

// PluginRegistry manages the registration and retrieval of scheduler plugins
type PluginRegistry struct {
	scorePluginBuilders  map[string]ScorePluginBuilder
	filterPluginBuilders map[string]FilterPluginBuilder
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		scorePluginBuilders:  make(map[string]ScorePluginBuilder),
		filterPluginBuilders: make(map[string]FilterPluginBuilder),
	}
}


// registerDefaultPlugins registers all default plugins to the given registry
func registerDefaultPlugins(registry *PluginRegistry) {
	// scorePlugin
	registry.registerScorePlugin(plugins.GPUCacheUsagePluginName, func(args runtime.RawExtension) framework.ScorePlugin {
		return plugins.NewGPUCacheUsage()
	})
	registry.registerScorePlugin(plugins.LeastLatencyPluginName, func(args runtime.RawExtension) framework.ScorePlugin {
		return plugins.NewLeastLatency(args)
	})
	registry.registerScorePlugin(plugins.LeastRequestPluginName, func(args runtime.RawExtension) framework.ScorePlugin {
		return plugins.NewLeastRequest(args)
	})
	registry.registerScorePlugin(plugins.RandomPluginName, func(args runtime.RawExtension) framework.ScorePlugin {
		return plugins.NewRandom(args)
	})
	// PrefixCache requires two parameters and is instantiated during use
	registry.registerScorePlugin(plugins.PrefixCachePluginName, func(args runtime.RawExtension) framework.ScorePlugin {
		return &plugins.PrefixCache{}
	})
	registry.registerScorePlugin(plugins.KVCacheAwarePluginName, func(args runtime.RawExtension) framework.ScorePlugin {
		return plugins.NewKVCacheAware(args)
	})

	// filterPlugin
	registry.registerFilterPlugin(plugins.LeastRequestPluginName, func(args runtime.RawExtension) framework.FilterPlugin {
		return plugins.NewLeastRequest(args)
	})
	registry.registerFilterPlugin(plugins.LoraAffinityPluginName, func(args runtime.RawExtension) framework.FilterPlugin {
		return plugins.NewLoraAffinity()
	})
}
