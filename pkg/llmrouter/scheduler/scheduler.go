package scheduler

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/llmrouter/datastore"
	"github.com/lx1036/gateway/pkg/llmrouter/scheduler/framework"
	"k8s.io/klog/v2"
	"sort"
	"time"
)

const (
	// Get the top five scoring podinfo
	topN = 5
)

type Scheduler interface {
	Schedule(ctx *framework.SchedulerContext, pods []*datastore.PodInfo) error
	RunPostHooks(ctx *framework.SchedulerContext, index int)
}

type FilterPlugin interface {
	Name() string
	// Filter is a method that is used to filter valid pods that can be sent request to.
	Filter(ctx *framework.SchedulerContext, pods []*datastore.PodInfo) []*datastore.PodInfo
}

type ScorePlugin interface {
	Name() string
	// Score is a method that is used to rank pods that have passed the filter plugins.
	// Note each plugin should generate score for a pod within [0, 100]
	Score(ctx *framework.SchedulerContext, pods []*datastore.PodInfo) map[*datastore.PodInfo]int
}

type scheduler struct {
	filterPlugins []FilterPlugin
	scorePlugins  []*scorePlugin
}

type scorePlugin struct {
	plugin ScorePlugin
	weight int
}

func NewScheduler() Scheduler {
	registry := NewPluginRegistry()
	registerDefaultPlugins(registry)

	return &scheduler{
		filterPlugins: getFilterPlugins(registry, filterPluginMap, pluginsArgMap),
		scorePlugins:  getScorePlugins(registry, prefixCache, scorePluginMap, pluginsArgMap),
	}
}

func (s *scheduler) Schedule(ctx *framework.SchedulerContext, pods []*datastore.PodInfo) error {
	// first filter out invalid pods that will not be selected to forward.
	pods, err := s.RunFilterPlugins(pods, ctx)
	if err != nil {
		return err
	}

	// INFO: PD 分离场景
	if ctx.PDGroup != nil {
		klog.Infof("schedule pods for Prefill-Decode")
		// TODO
	}

	klog.Infof("schedule pods for Non Prefill-Decode")
	scores := s.RunScorePlugins(pods, ctx)
	ctx.BestPods = TopNPodInfos(scores, topN)

	return nil
}

func (s *scheduler) RunPostHooks(ctx *framework.SchedulerContext, index int) {
	//TODO implement me
	panic("implement me")
}

func (s *scheduler) RunFilterPlugins(pods []*datastore.PodInfo, ctx *framework.SchedulerContext) ([]*datastore.PodInfo, error) {
	for _, filterPlugin := range s.filterPlugins {
		startTime := time.Now()
		// INFO: 这里 filter 后的 pods，作为下一个 plugin 的输入，单元测试没有问题
		pods = filterPlugin.Filter(ctx, pods)
		duration := time.Since(startTime)

		// Use the MetricsRecorder from context to record plugin duration
		if ctx.MetricsRecorder != nil {
			ctx.MetricsRecorder.RecordSchedulerPluginDuration(filterPlugin.Name(), metrics.PluginTypeFilter, duration)
		}

		if len(pods) == 0 {
			return nil, fmt.Errorf("pods have all been filtered out by %q", filterPlugin.Name())
		}
	}

	return pods, nil
}

func (s *scheduler) RunScorePlugins(pods []*datastore.PodInfo, ctx *framework.SchedulerContext) ([]*datastore.PodInfo, error) {
	res := make(map[*datastore.PodInfo]int)
	for _, plg := range s.scorePlugins {
		startTime := time.Now()
		scores := plg.plugin.Score(ctx, pods)
		duration := time.Since(startTime)

		// Use the MetricsRecorder from context to record plg duration
		if ctx.MetricsRecorder != nil {
			ctx.MetricsRecorder.RecordSchedulerPluginDuration(plg.Name(), metrics.PluginTypeFilter, duration)
		}

		if len(pods) == 0 {
			return nil, fmt.Errorf("pods have all been filtered out by %q", plg.Name())
		}

		for k, v := range scores {
			if k.Pod != nil {
				klog.Infof("Pod: %s/%s, Score: %d", k.Pod.Namespace, k.Pod.Name, v)
			}
			if _, ok := res[k]; !ok {
				res[k] = v * scorePlugin.weight
			} else {
				res[k] += v * scorePlugin.weight
			}
		}
	}

	return pods, nil
}

func TopNPodInfos(m map[*datastore.PodInfo]int, n int) []*datastore.PodInfo {
	var list []podInfoWithValue
	for k, v := range m {
		list = append(list, podInfoWithValue{pod: k, score: v})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].score > list[j].score
	})

	res := []*datastore.PodInfo{}
	for i := range list {
		if i >= n {
			break
		}
		res = append(res, list[i].pod)
	}

	return res
}
