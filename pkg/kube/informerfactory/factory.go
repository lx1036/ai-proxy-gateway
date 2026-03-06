package informerfactory

import (
	"github.com/lx1036/gateway/pkg/kube/kubetypes"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// InformerFactory provides access to a shared informer factory
type InformerFactory interface {
	// Start initializes all requested informers. They are handled in goroutines
	// which run until the stop channel gets closed.
	Start(stopCh <-chan struct{})

	// InformerFor returns the SharedIndexInformer the provided type.
	InformerFor(resource schema.GroupVersionResource, opts kubetypes.InformerOptions, newFunc NewInformerFunc) StartableInformer

	// WaitForCacheSync blocks until all started informers' caches were synced
	// or the stop channel gets closed.
	WaitForCacheSync(stopCh <-chan struct{}) bool

	// Shutdown marks a factory as shutting down. At that point no new
	// informers can be started anymore and Start will return without
	// doing anything.
	//
	// In addition, Shutdown blocks until all goroutines have terminated. For that
	// to happen, the close channel(s) that they were started with must be closed,
	// either before Shutdown gets called or while it is waiting.
	//
	// Shutdown may be called multiple times, even concurrently. All such calls will
	// block until all goroutines have terminated.
	Shutdown()
}

type NewInformerFunc func() cache.SharedIndexInformer

type StartableInformer struct {
	Informer cache.SharedIndexInformer
	start    func(stopCh <-chan struct{})
}

func (s StartableInformer) Start(stopCh <-chan struct{}) {
	s.start(stopCh)
}
