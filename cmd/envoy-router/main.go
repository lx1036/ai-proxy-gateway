package main

import (
	"flag"
	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/lx1036/gateway/pkg/router"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"net"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	grpcAddr string
)

func main() {
	flag.StringVar(&grpcAddr, "grpc-bind-address", ":50052", "The address the gRPC server binds to.")
	klog.InitFlags(flag.CommandLine)
	defer klog.Flush()
	flag.Parse()

	routerServer := router.NewServer()

	grpcServer := grpc.NewServer()
	extProcPb.RegisterExternalProcessorServer(grpcServer, routerServer)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}

	ctx := ctrl.SetupSignalHandler()
	go func() {
		<-ctx.Done()
		klog.Warningf("initiating graceful shutdown...")
		//routerServer.Shutdown()
		grpcServer.GracefulStop()
		os.Exit(0)
	}()

	if err := grpcServer.Serve(lis); err != nil {
		panic(err)
	}
}
