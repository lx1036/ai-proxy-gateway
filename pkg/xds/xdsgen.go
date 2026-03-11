package xds

import (
	"fmt"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/google/uuid"
	"github.com/lx1036/gateway/pkg/model"
	"github.com/lx1036/gateway/pkg/util"
	"strings"
)

func (s *DiscoveryServer) findGenerator(typeURL string, con *ConnectionContext) model.XdsResourceGenerator {

	if g, f := s.Generators[con.proxy.Metadata.Generator+"/"+typeURL]; f {
		return g
	}
	if g, f := s.Generators[string(con.proxy.Type)+"/"+typeURL]; f {
		return g
	}

	if g, f := s.Generators[typeURL]; f {
		return g
	}

	// XdsResourceGenerator is the default generator for this connection. We want to allow
	// some types to use custom generators - for example EDS.
	g := con.proxy.XdsResourceGenerator
	if g == nil {
		if strings.HasPrefix(typeURL, TypeDebugPrefix) {
			g = s.Generators["event"]
		} else {
			// TODO move this to just directly using the resource TypeUrl
			g = s.Generators["api"] // default to "MCP" generators - any type supported by store
		}
	}
	return g
}

// Push an XDS resource for the given connection. Configuration will be generated
// based on the passed in generator. Based on the updates field, generators may
// choose to send partial or even no response if there are no changes.
func (s *DiscoveryServer) pushXds(con *ConnectionContext, w *WatchedResource, req *model.PushRequest) error {
	if w == nil {
		return nil
	}
	gen := s.findGenerator(w.TypeUrl, con)
	if gen == nil {
		return nil
	}

	res, logdata, err := gen.Generate(con.proxy, w, req)

	resp := &discovery.DiscoveryResponse{
		ControlPlane: &corev3.ControlPlane{Identifier: fmt.Sprintf("%s-%s", w.TypeUrl, util.LookupEnv("POD_NAME", "defaultPod"))},
		TypeUrl:      w.TypeUrl,
		// TODO: send different version for incremental eds
		VersionInfo: req.Push.PushVersion,
		Nonce:       nonce(req.Push.PushVersion),
		Resources:   ResourcesToAny(res),
	}

	if err := Send(con, resp); err != nil {
		//if recordSendError(w.TypeUrl, err) {
		//	log.Warnf("%s: Send failure for node:%s resources:%d size:%s%s: %v",
		//		v3.GetShortType(w.TypeUrl), con.proxy.ID, len(res), util.ByteCount(configSize), info, err)
		//}
		return err
	}

	return nil
}

func nonce(noncePrefix string) string {
	return noncePrefix + uuid.New().String()
}
