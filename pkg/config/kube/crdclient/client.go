package crdclient

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/config"
	"github.com/lx1036/gateway/pkg/config/schema/collection"
)

type Option struct {
	Revision     string
	DomainSuffix string
	Identifier   string
	FiltersByGVK map[config.GroupVersionKind]kubetypes.Filter
	KrtDebugger  *krt.DebugHandler
}

type Client struct {
	schemas collection.Schemas
}

func NewForSchemas(client kube.Client, opts Option, schemas collection.Schemas) *Client {

	c := &Client{
		domainSuffix:     opts.DomainSuffix,
		schemas:          schemas,
		schemasByCRDName: schemasByCRDName,
		revision:         opts.Revision,
		started:          atomic.NewBool(false),
		kinds:            map[config.GroupVersionKind]nsStore{},
		client:           client,
		logger:           scope.WithLabels("controller", opts.Identifier),
		filtersByGVK:     opts.FiltersByGVK,
		stop:             stop,
	}

	kopts := krt.NewOptionsBuilder(stop, "crdclient", opts.KrtDebugger)
	for _, s := range c.schemas.All() {
		// From the spec: "Its name MUST be in the format <.spec.name>.<.spec.group>."
		name := fmt.Sprintf("%s.%s", s.Plural(), s.Group())
		c.addCRD(name, kopts)
	}

	return c
}
