package main

import (
	"github.com/lx1036/gateway/pkg/controllers/workloads/lws"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	leaderworkersetv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(leaderworkersetv1.AddToScheme(scheme))
}

func main() {
	kubeConfig := ctrl.GetConfigOrDie()
	options := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                server.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
	}
	mgr, err := ctrl.NewManager(kubeConfig, options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	go startControllers()

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func startControllers() {

	if err := lws.NewLeaderWorkerSetReconciler(mgr).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LeaderWorkerSet")
		os.Exit(1)
	}
}
