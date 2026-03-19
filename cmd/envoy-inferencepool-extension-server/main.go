package main

import (
	"flag"
	egextension "github.com/envoyproxy/gateway/proto/extension"
	"github.com/lx1036/gateway/pkg/extensionserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"k8s.io/klog/v2"
	"net"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	extensionServerPort := flag.String("port", "50052", "The port that the extension server listens on.")
	flag.Parse()

	ctx := ctrl.SetupSignalHandler()

	s := grpc.NewServer(grpc.MaxRecvMsgSize(4 * 1024 * 1024)) // 4MiB
	extSrv := extensionserver.New()

	egextension.RegisterEnvoyGatewayExtensionServer(s, extSrv)
	grpc_health_v1.RegisterHealthServer(s, extSrv)

	lis, err := net.Listen("tcp", *extensionServerPort)
	if err != nil {
		klog.ErrorS(err, "failed to listen", "port", *extensionServerPort)
		os.Exit(1)
	}

	go func() {
		if err := s.Serve(lis); err != nil {
			klog.Errorf("failed to serve extension server: %v", err)
		}
	}()

	<-ctx.Done()
	s.GracefulStop()
}
