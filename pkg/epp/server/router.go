package server

import (
	"context"
	"fmt"

	"github.com/lx1036/gateway/pkg/epp/datastore"
	"github.com/lx1036/gateway/pkg/epp/scheduling"
)

type Router struct {
	datastore *datastore.Datastore
	scheduler *scheduling.Scheduler
}

func (router *Router) HandleRequest(ctx context.Context, reqCtx *RequestContext) (*RequestContext, error) {

	var ok bool
	reqCtx.modelName, ok = reqCtx.Request.Body["model"].(string)
	if !ok {
		return reqCtx, fmt.Errorf("model not found in request body")
	}




	snapshotOfCandidatePods := router.datastore.GetPodsSnapshot()

	router.scheduler.Schedule(ctx, reqCtx.SchedulingRequest, snapshotOfCandidatePods)




	return reqCtx, nil
}
