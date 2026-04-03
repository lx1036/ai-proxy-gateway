package lws

import (
	"k8s.io/klog/v2"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)



func TestLeaderWorkerSetReconciler(test *testing.T) {
	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		klog.Fatalf("unable to start manager: %v", err)
		os.Exit(1)
	}

	err = NewLeaderWorkerSetReconciler(mgr).SetupWithManager(mgr)
	if err != nil {
		klog.Fatalf("unable to create controller: %v", err)
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("problem running manager: %v", err)
		os.Exit(1)
	}
}
