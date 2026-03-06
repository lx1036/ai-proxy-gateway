package typemap

import (
	"github.com/lx1036/gateway/pkg/ptr"
	"reflect"
)

// TypeMap provides a map that holds a map of Type -> Value. There can be only a single value per type.
// The value stored for a type must be of the same type as the key.
type TypeMap struct {
	inner map[reflect.Type]any
}

func NewTypeMap() TypeMap {
	return TypeMap{make(map[reflect.Type]any)}
}

func Set[T any](t TypeMap, v T) {
	t.inner[reflect.TypeFor[T]()] = v
}

func Get[T any](t TypeMap) *T {
	v, f := t.inner[reflect.TypeFor[T]()]
	if f {
		return ptr.Of(v.(T))
	}
	return nil
}
