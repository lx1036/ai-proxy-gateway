package istio

import (
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	"k8s.io/apimachinery/pkg/types"
)

type ClassController struct {
	queue controllers.Queue
}

func NewClassController(kc kube.Client) *ClassController {
	gc := &ClassController{}
	gc.queue = controllers.NewQueue("gateway class",
		controllers.WithReconciler(gc.Reconcile),
		controllers.WithMaxAttempts(25))
}

func (c *ClassController) Run(stop <-chan struct{}) {
	// Ensure we initially reconcile the current state
	c.queue.Add(types.NamespacedName{})
	c.queue.Run(stop)
}

func (c *ClassController) Reconcile(types.NamespacedName) error {

}
