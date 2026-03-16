package router

import (
	configPb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
)

func (s *Server) HandleRequestBody(request *extProcPb.ProcessingRequest) *extProcPb.ProcessingResponse {

	body := request.Request.(*extProcPb.ProcessingRequest_RequestBody).RequestBody.GetBody()

	var headers []*configPb.HeaderValueOption

	pods, err := s.cache.ListPodsByModel(model)
	if err != nil || len(pods) == 0 {
		return generateErrorResponse()
	}

	targetPodIP, err := s.selectTargetPod(pods)

	kvHeaders := map[string]string{
		HeaderTargetPod: targetPodIP,
	}
	headers = buildEnvoyProxyHeaders(headers, kvHeaders)

	return &extProcPb.ProcessingResponse{
		Response: &extProcPb.ProcessingResponse_RequestBody{
			RequestBody: &extProcPb.BodyResponse{
				Response: &extProcPb.CommonResponse{
					HeaderMutation: &extProcPb.HeaderMutation{
						SetHeaders: headers,
					},
					BodyMutation: &extProcPb.BodyMutation{
						Mutation: &extProcPb.BodyMutation_Body{
							Body: body,
						},
					},
				},
			},
		},
	}
}

func (s *Server) selectTargetPod() {
	readyPods := s.scheduler.Schedule()
}
