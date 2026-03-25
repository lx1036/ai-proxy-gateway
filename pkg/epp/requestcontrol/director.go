package requestcontrol

import (
	"context"
	"fmt"
)

type Scheduler interface {
	Schedule(ctx context.Context, request *schedulingtypes.LLMRequest, candidatePods []schedulingtypes.Pod) (result *schedulingtypes.SchedulingResult, err error)
}

// Director
// - Retrieving request metadata and relevant objectives.
// - Determining candidate pods.
// - Performing admission control via the AdmissionController.
// - Scheduling the request to target pod(s) via the Scheduler.
// - Running PreRequest plugins.
// - Preparing the request context for the Envoy ext_proc filter to route the request.
// - Running PostResponse plugins.
type Director struct {
	scheduler Scheduler
}

func NewDirectorWithConfig(scheduler Scheduler) *Director {
	return &Director{
		scheduler: scheduler,
	}
}

func (d *Director) HandleRequest(ctx context.Context, reqCtx *handlers.RequestContext) (*handlers.RequestContext, error) {

	// 1. admit request
	if err := d.admissionController.Admit(ctx, reqCtx, *infObjective.Spec.Priority); err != nil {
		logger.V(logutil.DEFAULT).Info("Request rejected by admission control", "error", err)
		return reqCtx, err
	}

	// 2. list candidate pods
	candidatePods := d.podLocator.Locate(ctx, reqCtx.Request.Metadata)
	if len(candidatePods) == 0 {
		return reqCtx, errutil.Error{
			Code: errutil.ServiceUnavailable,
			Msg:  "failed to find candidate pods for serving the request",
		}
	}

	// 3. schedule pods
	result, err := d.scheduler.Schedule(ctx, reqCtx.SchedulingRequest, snapshotOfCandidatePods)
	if err != nil {
		return reqCtx, errutil.Error{Code: errutil.InferencePoolResourceExhausted, Msg: fmt.Errorf("failed to find target pod: %w", err).Error()}
	}

}
