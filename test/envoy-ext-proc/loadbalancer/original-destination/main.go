package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"io"
	"k8s.io/klog/v2"
	"log"
	"net"
	"net/http"

	configPb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_api_v3_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_service_proc_v3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
)

type extProcServer struct{}

var (
	port int
)

func main() {
	flag.IntVar(&port, "port", 50051, "gRPC port")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	gs := grpc.NewServer()
	envoy_service_proc_v3.RegisterExternalProcessorServer(gs, &extProcServer{})

	go func() {
		err = gs.Serve(lis)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	http.HandleFunc("/healthz", healthCheckHandler)
	err = http.ListenAndServe(":7070", nil)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// used by k8s readiness probes
// makes a processing request to check if the processor service is healthy
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Create gRPC dial options
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}

	client := envoy_service_proc_v3.NewExternalProcessorClient(conn)
	processor, err := client.Process(context.Background())
	if err != nil {
		log.Fatalf("Could not check: %v", err)
	}

	err = processor.Send(&envoy_service_proc_v3.ProcessingRequest{
		Request: &envoy_service_proc_v3.ProcessingRequest_RequestHeaders{
			RequestHeaders: &envoy_service_proc_v3.HttpHeaders{},
		},
	})
	if err != nil {
		log.Fatalf("Could not check: %v", err)
	}

	response, err := processor.Recv()
	if err != nil {
		log.Fatalf("Could not check: %v", err)
	}

	if response != nil && response.GetRequestHeaders().Response.Status == envoy_service_proc_v3.CommonResponse_CONTINUE {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (s *extProcServer) Process(srv envoy_service_proc_v3.ExternalProcessor_ProcessServer) error {
	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := srv.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive stream request: %v", err)
		}

		resp := &envoy_service_proc_v3.ProcessingResponse{}
		switch v := req.Request.(type) {
		case *envoy_service_proc_v3.ProcessingRequest_RequestHeaders:
			xdsRouteName := ""

			if req.Attributes != nil {
				if epa, ok := req.Attributes["envoy.filters.http.ext_proc"]; ok {
					if rqa, ok := epa.Fields["xds.route_name"]; ok {
						xdsRouteName = rqa.GetStringValue()
					}
				}
			}

			emittedDynamicMetadata, _ := structpb.NewStruct(map[string]interface{}{
				"io.envoyproxy.gateway.e2e": map[string]interface{}{
					"ext-proc-emitted-metadata": "received",
				},
			})

			klog.Infof("xdsRouteName %s", xdsRouteName)

			headers := []*configPb.HeaderValueOption{
				{
					Header: &configPb.HeaderValue{
						Key: "target-pod",
						//Value: "127.0.0.1:8899",
						// python3 -m http.server 8899
						RawValue: []byte("127.0.0.1:8899"), // 必须 RawValue, Value 会报错 invalid headers
					},
				},
			}
			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_RequestHeaders{
					RequestHeaders: &envoy_service_proc_v3.HeadersResponse{
						Response: &envoy_service_proc_v3.CommonResponse{
							HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
								SetHeaders: headers,
							},
							ClearRouteCache: true,
						},
					},
				},
				DynamicMetadata: emittedDynamicMetadata,
			}

		case *envoy_service_proc_v3.ProcessingRequest_ResponseHeaders:
			respXDSRouteName := ""
			requestPath := ""
			if req.Attributes != nil {
				if epa, ok := req.Attributes["envoy.filters.http.ext_proc"]; ok {
					if rsa, ok := epa.Fields["xds.route_name"]; ok {
						respXDSRouteName = rsa.GetStringValue()
					}
					if rpa, ok := epa.Fields["request.path"]; ok {
						requestPath = rpa.GetStringValue()
					}
				}
			}

			forwardedDynamicMetadata := ""
			fmt.Printf("req: %+v\n", req)
			if req.MetadataContext != nil && req.MetadataContext.FilterMetadata != nil {
				if md, ok := req.MetadataContext.FilterMetadata["envoy.filters.http.rbac"]; ok {
					if mdf, ok := md.Fields["enforced_engine_result"]; ok {
						forwardedDynamicMetadata = mdf.GetStringValue()
					}
				}
			}

			rhq := &envoy_service_proc_v3.HeadersResponse{
				Response: &envoy_service_proc_v3.CommonResponse{
					HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
						SetHeaders: []*envoy_api_v3_core.HeaderValueOption{
							{
								Header: &envoy_api_v3_core.HeaderValue{
									Key:      "x-response-ext-processed",
									RawValue: []byte("true"),
								},
							},
							{
								Header: &envoy_api_v3_core.HeaderValue{
									Key:      "x-response-xds-route-name",
									RawValue: []byte(respXDSRouteName),
								},
							},
							{
								Header: &envoy_api_v3_core.HeaderValue{
									Key:      "x-response-rbac-result-metadata",
									RawValue: []byte(forwardedDynamicMetadata),
								},
							},
						},
					},
				},
			}

			if requestPath != "" {
				rhq.Response.HeaderMutation.SetHeaders = append(rhq.Response.HeaderMutation.SetHeaders,
					&envoy_api_v3_core.HeaderValueOption{
						Header: &envoy_api_v3_core.HeaderValue{
							Key:      "x-response-request-path",
							RawValue: []byte(requestPath),
						},
					})
			}

			headers := []*configPb.HeaderValueOption{
				{
					Header: &configPb.HeaderValue{
						Key: "target-pod",
						//Value: "127.0.0.1:8899",
						// python3 -m http.server 8899
						RawValue: []byte("127.0.0.1:8899"), // 必须 RawValue, Value 会报错 invalid headers
					},
				},
			}
			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_ResponseHeaders{
					ResponseHeaders: &envoy_service_proc_v3.HeadersResponse{
						Response: &envoy_service_proc_v3.CommonResponse{
							HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
								SetHeaders: headers,
							},
							ClearRouteCache: true,
						},
					},
				},
				//DynamicMetadata: &structpb.Struct{
				//	Fields: map[string]*structpb.Value{
				//		"io.envoyproxy.gateway.e2e": {
				//			Kind: &structpb.Value_StructValue{
				//				StructValue: &structpb.Struct{
				//					Fields: map[string]*structpb.Value{
				//						"request_cost_set_by_ext_proc": {
				//							Kind: &structpb.Value_NumberValue{NumberValue: float64(10)},
				//						},
				//					},
				//				},
				//			},
				//		},
				//	},
				//},
			}

		default:
			log.Printf("Unknown Request type %v\n", v)
		}

		if err := srv.Send(resp); err != nil {
			log.Printf("send error %v", err)
		}
	}
}
