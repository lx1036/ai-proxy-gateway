package plugins

import (
	"github.com/lx1036/gateway/pkg/llmrouter/datastore"
	"github.com/lx1036/gateway/pkg/llmrouter/scheduler/framework"
	"k8s.io/apimachinery/pkg/runtime"
)

const GPUCacheUsagePluginName = "gpu_cache_usage"

type GPUCacheUsage struct {
	name string
}

func NewGPUCacheUsage(pluginArg runtime.RawExtension) *GPUCacheUsage {


	return &GPUCacheUsage{
		name: GPUCacheUsagePluginName,
	}
}

func (r *GPUCacheUsage) Name() string {
	return r.name
}

// Score assigns random scores to pods within the range [0, 100]
func (r *GPUCacheUsage) Score(ctx *framework.SchedulerContext, pods []*datastore.PodInfo) map[*datastore.PodInfo]int {
	scoreResults := make(map[*datastore.PodInfo]int)
	for _, pod := range pods {
		// INFO: 简单粗暴，GPU cache usage 越小，得分越高
		scoreResults[pod] = int((1 - pod.GPUCacheUsage) * 100)
	}

	return scoreResults
}
