package crdclient

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/config"
	"github.com/lx1036/gateway/pkg/config/schema/collection"
	"github.com/lx1036/gateway/pkg/kube"
	"go.uber.org/atomic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

type Option struct {
	Revision     string
	DomainSuffix string
	Identifier   string
	FiltersByGVK map[config.GroupVersionKind]kubetypes.Filter
	KrtDebugger  *krt.DebugHandler
}

type nsStore struct {
	collection krt.Collection[config.Config]
	index      krt.Index[string, config.Config]
	handlers   []krt.HandlerRegistration
}

type Client struct {
	kindsMu sync.RWMutex

	schemas collection.Schemas

	// kinds keeps track of all cache handlers for known types
	kinds map[config.GroupVersionKind]nsStore
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

// Run the queue and all informers. Callers should wait for HasSynced() before depending on results.
func (cl *Client) Run(stop <-chan struct{}) {

	// TODO: 性能提升
	//if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced, modelInformer.HasSynced) {
	//	t.Error("timed out waiting for caches to sync")
	//}
	t0 := time.Now()
	if !kube.WaitForCacheSync("crdclient", stop, cl.informerSynced) {
		klog.Errorf("Failed to sync Pilot K8S CRD controller cache")
	} else {
		klog.Infof("Pilot K8S CRD controller synced in %v", time.Since(t0))
	}

	<-stop
	close(cl.stop)

	// cleanup
	for _, h := range cl.allKinds() {
		for _, reg := range h.handlers {
			reg.UnregisterHandler()
		}
	}
	klog.Infof("controller terminated")
}

func (cl *Client) informerSynced() bool {
	for gk, ctl := range cl.allKinds() {
		if !ctl.collection.HasSynced() {
			cl.logger.Infof("controller %q is syncing...", gk)
			return false
		}
	}
	return true
}

func (cl *Client) allKinds() map[config.GroupVersionKind]nsStore {
	cl.kindsMu.RLock()
	defer cl.kindsMu.RUnlock()
	return maps.Clone(cl.kinds)
}
