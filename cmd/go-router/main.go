package main

import (
	"github.com/lx1036/gateway/pkg/llmrouter/server"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {

	ctx := ctrl.SetupSignalHandler()

	server.NewServer().Run(ctx)

}
