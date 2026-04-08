package scheduling

import (
	"context"

	"github.com/lx1036/gateway/pkg/epp/scheduling/framework"
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

func (s *Scheduler) Schedule(ctx context.Context, request *framework.LLMRequest, candidatePods []framework.Pod) (result *framework.SchedulingResult, err error) {

	profileRunResults := map[string]*framework.ProfileRunResult{}
	cycleState := framework.NewCycleState()

	// get the next set of profiles to run iteratively based on the request and the previous execution results
	for {
		//profiles := s.profileHandler.Pick(ctx, cycleState, request, s.profiles, profileRunResults)
		profiles := s.profiles

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
