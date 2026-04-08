package server

import (
	"encoding/json"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"k8s.io/klog/v2"
	"time"

	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
)

type Request struct {
	Headers  map[string]string
	Body     map[string]any
	Metadata map[string]any
}

type Response struct {
	Headers map[string]string
}

type StreamRequestState int

const (
	RequestReceived                  StreamRequestState = 0
	HeaderRequestResponseComplete    StreamRequestState = 1
	BodyRequestResponsesComplete     StreamRequestState = 2
	TrailerRequestResponsesComplete  StreamRequestState = 3
	ResponseReceived                 StreamRequestState = 4
	HeaderResponseResponseComplete   StreamRequestState = 5
	BodyResponseResponsesComplete    StreamRequestState = 6
	TrailerResponseResponsesComplete StreamRequestState = 7
)

// RequestContext stores context information during the life time of an HTTP request.
type RequestContext struct {
	modelName string

	Request                   *Request
	RequestState              StreamRequestState
	RequestReceivedTimestamp  time.Time
	ResponseCompleteTimestamp time.Time

	Response *Response
}

// ExtProcServer implements the Envoy external processing server.
// https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/ext_proc/v3/external_processor.proto
type ExtProcServer struct {
	extProcPb.UnimplementedExternalProcessorServer

	router *router.Router
}

func NewExtProcServer(datastore Datastore, director Director) *ExtProcServer {
	return &ExtProcServer{
		router:    director,
		datastore: datastore,
	}
}

func (s *ExtProcServer) Process(srv extProcPb.ExternalProcessor_ProcessServer) error {

	ctx := srv.Context()

	// Create request context to share states during life time of an HTTP request.
	// See https://github.com/envoyproxy/envoy/issues/17540.
	// 值得一看
	reqCtx := &RequestContext{
		RequestState: RequestReceived,
		Request: &Request{
			Headers:  make(map[string]string),
			Body:     make(map[string]any),
			Metadata: make(map[string]any),
		},
		Response: &Response{
			Headers: make(map[string]string),
		},
	}

	var body []byte
	var err error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, recvErr := srv.Recv()
		if recvErr == io.EOF || status.Code(recvErr) == codes.Canceled {
			return nil
		}
		if recvErr != nil {
			return status.Errorf(codes.Unknown, "cannot receive stream request: %v", err)
		}

		switch v := req.Request.(type) {
		case *extProcPb.ProcessingRequest_RequestHeaders:
			requestID := ExtractHeadersValue(v, RequestIdHeaderKey)
			if len(requestID) == 0 {
				requestID = uuid.NewString()
				klog.Infof("RequestID header is not found in the request, generated a request id")
				reqCtx.Request.Headers[RequestIdHeaderKey] = requestID
			}

			err = s.HandleRequestHeaders(reqCtx, v)

		case *extProcPb.ProcessingRequest_RequestBody:

			klog.InfoS("Incoming body chunk", "EoS", v.RequestBody.EndOfStream)
			// In the stream case, we can receive multiple request bodies.
			body = append(body, v.RequestBody.Body...)

			// INFO: 考虑了 stream 的情况，会收到多个 body chunk。不过 aibrix 里也没有去考虑，直接用 v.RequestBody.Body
			// Message is buffered, we can read and decode.
			if v.RequestBody.EndOfStream {
				klog.Infof("decoding request body...")

				if errUnmarshal := json.Unmarshal(body, &reqCtx.Request.Body); errUnmarshal != nil {
					klog.Errorf("cannot unmarshal request body: %v", errUnmarshal)
					err = status.Errorf(codes.Unknown, "cannot unmarshal request body: %v", errUnmarshal)
					break
				}

				// Body stream complete. Allocate empty slice for response to use.
				body = []byte{}

				reqCtx, err = s.router.HandleRequest(ctx, reqCtx)
				if err != nil {
					logger.V(logutil.DEFAULT).Error(err, "Error handling request")
					break
				}

				// Populate the ExtProc protocol responses for the request body.
				requestBodyBytes, err := json.Marshal(reqCtx.Request.Body)
				if err != nil {
					klog.Errorf("cannot marshal request body error: %v", err)
					break
				}
				reqCtx.RequestSize = len(requestBodyBytes)
				reqCtx.reqHeaderResp = s.generateRequestHeaderResponse(reqCtx)
				reqCtx.reqBodyResp = s.generateRequestBodyResponses(requestBodyBytes)

			}

		case *extProcPb.ProcessingRequest_RequestTrailers:

		case *extProcPb.ProcessingRequest_ResponseHeaders:

		case *extProcPb.ProcessingRequest_ResponseBody:

			if reqCtx.modelServerStreaming {
				// Currently we put on response parsing if the modelServer is streaming, and we just passthrough.

			} else {

			}

		case *extProcPb.ProcessingRequest_ResponseTrailers:

		}

		if err != nil {
			err := s.generateErrorResponse(reqCtx, err)
		}

		if err := srv.Send(resp); err != nil {
			return status.Errorf(codes.Unknown, "cannot send stream response: %v", err)
		}
	}

}
