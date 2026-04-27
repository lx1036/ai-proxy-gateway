package infra

import (
	"context"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"github.com/lx1036/gateway/pkg/envoygateway/message"
	"github.com/lx1036/gateway/pkg/envoygateway/scheme"
	"github.com/telepresenceio/watchable"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Infra struct {
	InfraClient *InfraClient

	InfraIR *message.InfraIR
}

func NewInfra(infraIR *message.InfraIR) (*Infra, error) {
	restCfg := ctrl.GetConfigOrDie()
	k8sClient, err := client.New(restCfg, client.Options{
		HTTPClient:      nil,
		Scheme:          scheme.GetScheme(),
		Mapper:          nil,
		Cache:           nil,
		DryRun:          nil,
		FieldOwner:      "",
		FieldValidation: "",
	})
	if err != nil {
		return nil, err
	}

	return &Infra{
		InfraClient: NewInfraClient(k8sClient),
		InfraIR: infraIR,
	}, nil
}

func (infra *Infra) Start(ctx context.Context) error {

	sub := infra.InfraIR.Subscribe(ctx)
	go message.HandleSubscription(sub, func(update watchable.Update[string, *ir.Infra]) {
		val := update.Value
		if update.Delete { // delete

		} else { // create or update

			infra.CreateOrUpdateProxyInfra(ctx, val)
		}
	})

	return nil
}

