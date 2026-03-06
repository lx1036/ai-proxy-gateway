package kubetypes

import (
	"github.com/lx1036/gateway/pkg/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type RegisterType[T runtime.Object] interface {
	GetGVK() config.GroupVersionKind
	GetGVR() schema.GroupVersionResource
	Object() T
}
