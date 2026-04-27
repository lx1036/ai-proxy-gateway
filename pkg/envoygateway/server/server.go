package server

import (
	"context"
	"github.com/lx1036/gateway/pkg/envoygateway/gatewayapi"
	"github.com/lx1036/gateway/pkg/envoygateway/infra"
	"github.com/lx1036/gateway/pkg/envoygateway/message"
	"github.com/lx1036/gateway/pkg/envoygateway/provider/kubernetes"
)

func startRunners(ctx context.Context)  {


	providerResources := new(message.ProviderResources)
	infraIR := new(message.InfraIR)
	xdsIR := new(message.XdsIR)

	provider, err := kubernetes.NewProvider(ctx, providerResources)



	translator, err := gatewayapi.NewTranslator(infraIR, xdsIR)


	ifr, err := infra.NewInfra(infraIR)
}


