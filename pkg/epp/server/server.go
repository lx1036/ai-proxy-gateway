package server

import (
	"context"
	"fmt"
	extProcPb "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"net"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ExtProcRunner
// 1. reconcile Pod/InferencePool
// 2. start ext-proc server
type ExtProcRunner struct {
}

// SetupWithManager reconcile Pod/InferencePool
func (r *ExtProcRunner) SetupWithManager(mgr ctrl.Manager) error {
	if err := (&controller.InferencePoolReconciler{
		Datastore: r.Datastore,
		Reader:    mgr.GetClient(),
		PoolGKNN:  r.GKNN,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed setting up InferencePoolReconciler - %w", err)
	}

	if err := (&controller.PodReconciler{
		Datastore: r.Datastore,
		Reader:    mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed setting up PodReconciler - %w", err)
	}

	return nil
}

// AsRunnable start ext-proc gRPC server.
// 使用 controller-runtime manager.Runnable 函数创建 Runnable
func (r *ExtProcRunner) AsRunnable() manager.Runnable {
	return manager.RunnableFunc(func(ctx context.Context) error {
		grpcServer := grpc.NewServer()
		extProcServer := NewExtProcServer(r.Datastore, r.Director)
		extProcPb.RegisterExternalProcessorServer(grpcServer, extProcServer)

		// Start listening.
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return fmt.Errorf("gRPC server failed to listen - %w", err)
		}

		klog.InfoS("gRPC server listening", "port", port)

		ctxStop := ctrl.SetupSignalHandler()
		go func() {
			<-ctxStop.Done()
			klog.Warningf("initiating graceful shutdown...")
			grpcServer.GracefulStop()
			os.Exit(0)
		}()

		// Keep serving until terminated.
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			return fmt.Errorf("gRPC server failed - %w", err)
		}
		klog.Info("gRPC server terminated")
		return nil
	})
}
