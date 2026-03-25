package scheduling

import (
	"context"

	"github.com/lx1036/gateway/pkg/epp/scheduling/framework"
	"github.com/lx1036/gateway/pkg/epp/scheduling/types"
)

type SchedulerConfig struct {
	profileHandler framework.ProfileHandler
	profiles       map[string]*framework.SchedulerProfile
}

type Scheduler struct {
	profileHandler framework.ProfileHandler
	profiles       map[string]*framework.SchedulerProfile
}

func NewSchedulerWithConfig(config *SchedulerConfig) *Scheduler {
	return &Scheduler{
		profileHandler: config.profileHandler,
		profiles:       config.profiles,
	}
}

func (s *Scheduler) Schedule(ctx context.Context, request *types.LLMRequest, candidatePods []types.Pod) (result *types.SchedulingResult, err error) {

	profileRunResults := map[string]*types.ProfileRunResult{}
	cycleState := types.NewCycleState()

	// get the next set of profiles to run iteratively based on the request and the previous execution results
	for {
		profiles := s.profileHandler.Pick(ctx, cycleState, request, s.profiles, profileRunResults)

		if len(profiles) == 0 { // profile picker didn't pick any profile to run
			break
		}

		for name, profile := range profiles {
			profileRunResult, err := profile.Run(ctx, request, cycleState, candidatePods)

		}

		profileRunResults[name] = profileRunResult
	}

	result, err = s.profileHandler.ProcessResults(ctx, cycleState, request, profileRunResults)

	return result, err
}
