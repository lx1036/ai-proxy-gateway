package lws

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	leaderworkersetv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
	"testing"
)

// leader Pod owner -> worker sts

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(leaderworkersetv1.AddToScheme(scheme))
}

func TestStatefulSetReconciler(test *testing.T) {
	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		klog.Fatalf("unable to start manager: %v", err)
		os.Exit(1)
	}

	err = NewWorkerStatefulSetReconciler(mgr).SetupWithManager(mgr)
	if err != nil {
		klog.Fatalf("unable to create controller: %v", err)
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("problem running manager: %v", err)
		os.Exit(1)
	}
}
