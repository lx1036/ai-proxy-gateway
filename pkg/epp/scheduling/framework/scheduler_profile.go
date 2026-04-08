package framework

import (
	"context"
	"fmt"
)

// SchedulerProfile provides a profile configuration for the scheduler which influence routing decisions.
type SchedulerProfile struct {
	filters []Filter
	scorers []*WeightedScorer
	picker  Picker
}

// Run filter -> scorer -> picker
func (p *SchedulerProfile) Run(ctx context.Context, request *LLMRequest, cycleState *CycleState, candidatePods []Pod) (*ProfileRunResult, error) {
	pods := p.runFilterPlugins(ctx, request, cycleState, candidatePods)
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods available for the given request")
	}
	// if we got here, there is at least one pod to score
	weightedScorePerPod := p.runScorerPlugins(ctx, request, cycleState, pods)

	result := p.runPickerPlugin(ctx, cycleState, weightedScorePerPod)

	return result, nil
}

func (p *SchedulerProfile) runFilterPlugins(ctx context.Context, request *LLMRequest, cycleState *CycleState, pods []Pod) []Pod {
	filteredPods := pods
	
	for _, filter := range p.filters {
		filteredPods = filter.Filter(ctx, cycleState, request, filteredPods)


		if len(filteredPods) == 0 {
			break
		}
	}

	return filteredPods
}

func (p *SchedulerProfile) runScorerPlugins(ctx context.Context, request *LLMRequest, cycleState *CycleState, pods []Pod) map[Pod]float64 {
	
}


func (p *SchedulerProfile) runPickerPlugin(ctx context.Context, cycleState *CycleState, weightedScorePerPod map[Pod]float64) *ProfileRunResult {
	scoredPods := make([]*ScoredPod, len(weightedScorePerPod))
	i := 0
	for pod, score := range weightedScorePerPod {
		scoredPods[i] = &ScoredPod{Pod: pod, Score: score}
		i++
	}

	result := p.picker.Pick(ctx, cycleState, scoredPods)



	return result
}
