package pd

import (
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
)

var (
	Connectors = map[string]func() KVConnector{}
)

func init() {
	Register("lmcache", NewLMCacheConnector)
	Register("nixl", NewNixlConnector)
	Register("mooncake", NewMoonCakeConnector)
}

type KVConnector interface {
	Proxy(c *gin.Context, modelRequest map[string]interface{}, prefillEndpoint, decodeEndpoint string)
}

func Register(name string, connector func() KVConnector) {
	connector, ok := Connectors[name]
	if ok {
		klog.Fatalf("connector %s is existed", name)
	}

	Connectors[name] = connector
}

func GetConnector(name string) func() KVConnector {
	connector, ok := Connectors[name]
	if !ok {
		klog.Fatalf("connector %s is not existed", name)
	}
	return connector
}
