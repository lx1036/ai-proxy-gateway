package router

import (
	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/lx1036/gateway/pkg/cache"
	"k8s.io/klog/v2"
)

type Server struct {
	cache cache.Cache
}

func NewServer() *Server {

	c := cache.NewCache()

	return &Server{
		cache: c,
	}
}

func (s *Server) Process(stream extProcPb.ExternalProcessor_ProcessServer) error {
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		klog.Infof("processing request ")

		request, err := stream.Recv()
		if err != nil {
			klog.Errorf("error receiving request %v", err)
			return err
		}

		response, err := s.handleProcessingRequest(request)

		return s.sendProcessingResponse(stream, response)
	}
}

func (s *Server) handleProcessingRequest(request *extProcPb.ProcessingRequest) (*extProcPb.ProcessingResponse, error) {
	var resp *extProcPb.ProcessingResponse

	switch request.Request.(type) {
	case *extProcPb.ProcessingRequest_RequestHeaders:
		resp = s.HandleRequestHeaders(request)

	case *extProcPb.ProcessingRequest_RequestBody:
		resp = s.HandleRequestBody(request)

	case *extProcPb.ProcessingRequest_ResponseHeaders:

	case *extProcPb.ProcessingRequest_ResponseBody:

	}

	return resp, nil
}

func (s *Server) sendProcessingResponse(stream extProcPb.ExternalProcessor_ProcessServer, response *extProcPb.ProcessingResponse) error {
	if err := stream.Send(response); err != nil {
		klog.Errorf("error sending response %v", err)
		return err
	}

	return nil
}
