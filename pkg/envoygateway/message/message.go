package message

import (
	"github.com/lx1036/gateway/pkg/envoygateway/gatewayapi/resource"

	"github.com/telepresenceio/watchable"
)

// ProviderResources INFO: 用于各个模块代码的 消息订阅模式
type ProviderResources struct {
	GatewayAPIResources watchable.Map[string, *resource.ControllerResources]
}
