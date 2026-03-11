package common

import (
	"istio.io/istio/pilot/pkg/model"
)

type GatewayController interface {
	model.ConfigStoreController
}
