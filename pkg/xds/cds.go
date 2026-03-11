package xds

import (
	"github.com/lx1036/gateway/pkg/model"
	"github.com/lx1036/gateway/pkg/networking/core"
)

type CdsGenerator struct {
	ConfigGenerator *core.ConfigGenerator
}

func (c CdsGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	req, needsPush := cdsNeedsPush(req, proxy)
	if !needsPush {
		return nil, model.DefaultXdsLogDetails, nil
	}
	clusters, logs := c.ConfigGenerator.BuildClusters(proxy, req)
	return clusters, logs, nil
}
