package framework

import (
	"context"

	"github.com/lx1036/gateway/pkg/epp/plugins"
)

/**
https://github.com/kubernetes-sigs/gateway-api-inference-extension/blob/main/docs/proposals/0845-scheduler-architecture-proposal/README.md
*/

// ProfileHandler defines the extension points for handling multi SchedulerProfile instances.
type ProfileHandler interface {
	plugins.Plugin

	Pick(ctx context.Context, cycleState *CycleState, request *LLMRequest, profiles map[string]*SchedulerProfile,
		profileResults map[string]*ProfileRunResult) map[string]*SchedulerProfile

	ProcessResults(ctx context.Context, cycleState *CycleState, request *LLMRequest,
		profileResults map[string]*ProfileRunResult) (*SchedulingResult, error)
}

type Filter interface {
	plugins.Plugin
	Filter(ctx context.Context, cycleState *CycleState, request *LLMRequest, pods []Pod) []Pod
}

type Scorer interface {
	plugins.Plugin
	Score(ctx context.Context, cycleState *CycleState, request *LLMRequest, pods []Pod) map[Pod]float64
}

type WeightedScorer struct {
	Scorer
	weight int
}

// Weight returns the weight of the scorer.
func (s *WeightedScorer) Weight() int {
	return s.weight
}

// Picker picks the final pod(s) to send the request to.
type Picker interface {
	plugins.Plugin
	Pick(ctx context.Context, cycleState *CycleState, scoredPods []*ScoredPod) *ProfileRunResult
}
