package kubetypes

import (
	"context"
	"k8s.io/apimachinery/pkg/watch"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type InformerOptions struct {
	// A selector to restrict the list of returned objects by their labels.
	LabelSelector string
	// A selector to restrict the list of returned objects by their fields.
	FieldSelector string
	// Namespace to watch.
	Namespace string
	// Cluster name for watch error handlers
	//Cluster cluster.ID
	// ObjectTransform allows arbitrarily modifying objects stored in the underlying cache.
	// If unset, a default transform is provided to remove ManagedFields (high cost, low value)
	ObjectTransform func(obj any) (any, error)
	// InformerType dictates the type of informer that should be created.
	InformerType InformerType
}

type InformerType int

const (
	StandardInformer InformerType = iota
	DynamicInformer
	MetadataInformer
)

// WriteAPI exposes a generic API for a client go type for write operations.
type WriteAPI[T runtime.Object] interface {
	Create(ctx context.Context, object T, opts metav1.CreateOptions) (T, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result T, err error)
	Update(ctx context.Context, object T, opts metav1.UpdateOptions) (T, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

// ReadAPI exposes a generic API for a client go type for read operations.
type ReadAPI[T runtime.Object, TL runtime.Object] interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (T, error)
	List(ctx context.Context, opts metav1.ListOptions) (TL, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

// ReadWriteAPI exposes a generic API for read and write operations.
type ReadWriteAPI[T runtime.Object, TL runtime.Object] interface {
	ReadAPI[T, TL]
	WriteAPI[T]
}
