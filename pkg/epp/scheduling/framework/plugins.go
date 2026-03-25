package framework

import (
	"context"

	"github.com/lx1036/gateway/pkg/epp/plugins"
	"github.com/lx1036/gateway/pkg/epp/scheduling/types"
)

/**
https://github.com/kubernetes-sigs/gateway-api-inference-extension/blob/main/docs/proposals/0845-scheduler-architecture-proposal/README.md
*/

// ProfileHandler defines the extension points for handling multi SchedulerProfile instances.
type ProfileHandler interface {
	plugins.Plugin

	Pick(ctx context.Context, cycleState *types.CycleState, request *types.LLMRequest, profiles map[string]*SchedulerProfile,
		profileResults map[string]*types.ProfileRunResult) map[string]*SchedulerProfile

	ProcessResults(ctx context.Context, cycleState *types.CycleState, request *types.LLMRequest,
		profileResults map[string]*types.ProfileRunResult) (*types.SchedulingResult, error)
}

type Filter interface {
	plugins.Plugin
	Filter(ctx context.Context, cycleState *types.CycleState, request *types.LLMRequest, pods []types.Pod) []types.Pod
}

type Scorer interface {
	plugins.Plugin
	Score(ctx context.Context, cycleState *types.CycleState, request *types.LLMRequest, pods []types.Pod) map[types.Pod]float64
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
	Pick(ctx context.Context, cycleState *types.CycleState, scoredPods []*types.ScoredPod) *types.ProfileRunResult
}
