package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	envoy_api_v3_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_service_proc_v3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type extProcServer struct {
	envoy_service_proc_v3.UnimplementedExternalProcessorServer
}

var (
	port     int
	certPath string
)

const (
	DefaultDestinationEndpointHintMetadataNamespace = "envoy.lb"                       // default for --destinationEndpointHintMetadataNamespace
	DefaultDestinationEndpointHintKey               = "x-gateway-destination-endpoint" // default for --destinationEndpointHintKey
)

// go run .
func main2() {
	flag.IntVar(&port, "port", 9000, "gRPC port")
	flag.StringVar(&certPath, "certPath", "", "path to extProcServer certificate and private key")
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

	// Create Unix listener
	gus := grpc.NewServer()
	envoy_service_proc_v3.RegisterExternalProcessorServer(gus, &extProcServer{})

	udsAddr, err := filepath.Abs("./extproc.sock")
	if err != nil {
		log.Fatalf("failed to abs file error: %v", err)
	}
	//udsAddr := "/tmp/extproc.sock"
	if _, err := os.Stat(udsAddr); err == nil {
		if err := os.RemoveAll(udsAddr); err != nil {
			log.Fatalf("failed to remove: %v", err)
		}
	}

	ul, err := net.Listen("unix", udsAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	err = os.Chmod(udsAddr, 0o700)
	if err != nil {
		log.Fatalf("failed to set permissions: %v", err)
	}

	// envoy distroless uid
	//err = os.Chown(udsAddr, 65532, 0)
	//if err != nil {
	//	log.Fatalf("failed to set permissions: %v", err)
	//}

	go func() {
		err = gus.Serve(ul)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	log.Printf("listen and serve ext-proc grpc server at :%d successfully", port)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *extProcServer) Process2(srv envoy_service_proc_v3.ExternalProcessor_ProcessServer) error {
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

			xrch := ""
			if v.RequestHeaders != nil {
				hdrs := v.RequestHeaders.Headers.GetHeaders()
				for _, hdr := range hdrs {
					if hdr.Key == "x-request-client-header" {
						xrch = string(hdr.RawValue)
					}
				}
			}

			rhq := &envoy_service_proc_v3.HeadersResponse{
				Response: &envoy_service_proc_v3.CommonResponse{
					HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
						SetHeaders: []*envoy_api_v3_core.HeaderValueOption{
							{
								Header: &envoy_api_v3_core.HeaderValue{
									Key:      "x-request-ext-processed",
									RawValue: []byte("true"),
								},
							},
							{
								Header: &envoy_api_v3_core.HeaderValue{
									Key:      "x-request-xds-route-name",
									RawValue: []byte(xdsRouteName),
								},
							},
						},
					},
				},
			}

			if xrch != "" {
				rhq.Response.HeaderMutation.SetHeaders = append(rhq.Response.HeaderMutation.SetHeaders,
					&envoy_api_v3_core.HeaderValueOption{
						Header: &envoy_api_v3_core.HeaderValue{
							Key:      "x-request-client-header",
							RawValue: []byte("mutated"),
						},
					})
				rhq.Response.HeaderMutation.SetHeaders = append(rhq.Response.HeaderMutation.SetHeaders,
					&envoy_api_v3_core.HeaderValueOption{
						Header: &envoy_api_v3_core.HeaderValue{
							Key:      "x-request-client-header-received",
							RawValue: []byte(xrch),
						},
					})
			}

			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_RequestHeaders{
					RequestHeaders: rhq,
				},
				DynamicMetadata: emittedDynamicMetadata,
			}

			break
		case *envoy_service_proc_v3.ProcessingRequest_ResponseHeaders:

			respXDSRouteName := ""

			if req.Attributes != nil {
				if epa, ok := req.Attributes["envoy.filters.http.ext_proc"]; ok {
					if rsa, ok := epa.Fields["xds.route_name"]; ok {
						respXDSRouteName = rsa.GetStringValue()
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

			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_ResponseHeaders{
					ResponseHeaders: rhq,
				},
				DynamicMetadata: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"io.envoyproxy.gateway.e2e": {
							Kind: &structpb.Value_StructValue{
								StructValue: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"request_cost_set_by_ext_proc": {
											Kind: &structpb.Value_NumberValue{NumberValue: float64(10)},
										},
									},
								},
							},
						},
					},
				},
			}
			break
		default:
			log.Printf("Unknown Request type %v\n", v)
		}

		if err := srv.Send(resp); err != nil {
			log.Printf("send error %v", err)
		}
	}
}

// Process handles external processing requests from Envoy.
// It listens for incoming requests, modifies response headers,
// and sends the updated response back to Envoy.
//
// When a request with response headers is received, it adds a custom header
// "x-extproc-hello" with the value "Hello from ext_proc" and returns the modified headers.
//
// Note: The `RawValue` field is used instead of `Value` because it supports
// setting the header value as a byte slice, allowing precise handling of binary data.
//
// This function is called once per HTTP request to process gRPC messages from Envoy.
// It exits when an error occurs while receiving or sending messages.

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
			// all headers
			var headers []*envoy_api_v3_core.HeaderValueOption
			for _, header := range v.RequestHeaders.Headers.Headers {
				if header.RawValue != nil {
					headers = append(headers, &envoy_api_v3_core.HeaderValueOption{
						Header: &envoy_api_v3_core.HeaderValue{
							Key:      header.Key,
							RawValue: []byte(header.RawValue),
						},
					})
				} else {
					headers = append(headers, &envoy_api_v3_core.HeaderValueOption{
						Header: &envoy_api_v3_core.HeaderValue{
							Key:      header.Key,
							RawValue: []byte(header.Value),
						},
					})
				}
			}

			endpoint := "127.0.0.1:7788" // python3 -m http.server 5566
			headers = append(headers, &envoy_api_v3_core.HeaderValueOption{
				Header: &envoy_api_v3_core.HeaderValue{
					Key:      DefaultDestinationEndpointHintKey,
					RawValue: []byte(endpoint),
				},
			})

			// ok!!!
			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_RequestHeaders{
					RequestHeaders: &envoy_service_proc_v3.HeadersResponse{
						Response: &envoy_service_proc_v3.CommonResponse{
							ClearRouteCache: true,
							HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
								SetHeaders: headers,
							},
						},
					},
				},
				DynamicMetadata: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						DefaultDestinationEndpointHintMetadataNamespace: {
							Kind: &structpb.Value_StructValue{
								StructValue: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										DefaultDestinationEndpointHintKey: {
											Kind: &structpb.Value_StringValue{
												StringValue: endpoint,
											},
										},
									},
								},
							},
						},
					},
				},
			}

			// not ok!!!
			// envoy.lb 没起作用

			// Build response according to EPP specification:
			// Set x-gateway-destination-endpoint in dynamic metadata (namespace: envoy.lb)
			// to communicate selected endpoint to data plane

			//lbMetadata, _ := structpb.NewStruct(map[string]interface{}{
			//	"x-gateway-destination-endpoint": endpoint,
			//})
			//dynamicMetadata, _ := structpb.NewStruct(map[string]interface{}{
			//	"envoy.lb": lbMetadata.AsMap(),
			//})
			//resp = &envoy_service_proc_v3.ProcessingResponse{
			//	Response: &envoy_service_proc_v3.ProcessingResponse_RequestHeaders{
			//		RequestHeaders: &envoy_service_proc_v3.HeadersResponse{
			//			Response: &envoy_service_proc_v3.CommonResponse{
			//				HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
			//					RemoveHeaders: []string{"x-endpoint"},
			//				},
			//			},
			//		},
			//	},
			//	DynamicMetadata: dynamicMetadata,
			//}

			log.Printf("Sending request headers: %+v\n", resp)
			break

		case *envoy_service_proc_v3.ProcessingRequest_RequestBody:
			break
		case *envoy_service_proc_v3.ProcessingRequest_RequestTrailers:
			break

		case *envoy_service_proc_v3.ProcessingRequest_ResponseHeaders:
			log.Println("Processing Response Headers...")

			resp = &envoy_service_proc_v3.ProcessingResponse{
				Response: &envoy_service_proc_v3.ProcessingResponse_ResponseHeaders{
					ResponseHeaders: &envoy_service_proc_v3.HeadersResponse{
						Response: &envoy_service_proc_v3.CommonResponse{
							HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
								SetHeaders: []*envoy_api_v3_core.HeaderValueOption{
									{
										Header: &envoy_api_v3_core.HeaderValue{
											Key:      "x-extproc-hello",
											RawValue: []byte("Hello from ext_proc"),
										},
									},
								},
							},
						},
					},
				},
			}
			log.Printf("Sending response headers: %+v\n", resp)
			break

		case *envoy_service_proc_v3.ProcessingRequest_ResponseBody:
			break
		case *envoy_service_proc_v3.ProcessingRequest_ResponseTrailers:
			break
		default:
			log.Printf("Unknown Request type %v\n", v)
		}

		if err := srv.Send(resp); err != nil {
			log.Printf("send error %v", err)
		}
	}

	//for {
	//	req, err := srv.Recv()
	//	if err != nil {
	//		return status.Errorf(codes.Unknown, "error receiving request: %v", err)
	//	}
	//
	//	log.Printf("Received request: %+v\n", req)
	//
	//	// Prepare the response to be returned to Envoy.
	//	resp := &envoy_service_proc_v3.ProcessingResponse{}
	//
	//	// Only process response headers, not request headers.
	//	if respHeaders := req.GetResponseHeaders(); respHeaders != nil {
	//		log.Println("Processing Response Headers...")
	//
	//		resp = &envoy_service_proc_v3.ProcessingResponse{
	//			Response: &envoy_service_proc_v3.ProcessingResponse_ResponseHeaders{
	//				ResponseHeaders: &envoy_service_proc_v3.HeadersResponse{
	//					Response: &envoy_service_proc_v3.CommonResponse{
	//						HeaderMutation: &envoy_service_proc_v3.HeaderMutation{
	//							SetHeaders: []*envoy_api_v3_core.HeaderValueOption{
	//								{
	//									Header: &envoy_api_v3_core.HeaderValue{
	//										Key:      "x-extproc-hello",
	//										RawValue: []byte("Hello from ext_proc"),
	//									},
	//								},
	//							},
	//						},
	//					},
	//				},
	//			},
	//		}
	//		log.Printf("Sending response: %+v\n", resp)
	//		// Send the response back to Envoy.
	//		if err := srv.Send(resp); err != nil {
	//			return status.Errorf(codes.Unknown, "error sending response: %v", err)
	//		}
	//	} else {
	//		// If it is not a callback in the response header stage, do not make any modifications and continue processing the next event.
	//		// For request_headers or other events, do not modify & ensure that Envoy will not be stuck.
	//		// An empty processing can be returned for request_headers, or it can be skipped in envoy.yaml.
	//		// Here, simply continue to wait for the next event.
	//		continue
	//	}
	//}
}

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	// Register the ExternalProcessorServer with the gRPC server.
	envoy_service_proc_v3.RegisterExternalProcessorServer(grpcServer, &extProcServer{})

	log.Println("Starting gRPC server on :9000...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
