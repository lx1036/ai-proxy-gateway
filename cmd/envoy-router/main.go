package main

import (
	"flag"
	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/lx1036/gateway/pkg/router"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"net"
	"os"
	"os/signal"
	"syscall"
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

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-gracefulStop
		klog.Warningf("signal received: %v, initiating graceful shutdown...", sig)
		//routerServer.Shutdown()
		grpcServer.GracefulStop()
		os.Exit(0)
	}()

	if err := grpcServer.Serve(lis); err != nil {
		panic(err)
	}
}
