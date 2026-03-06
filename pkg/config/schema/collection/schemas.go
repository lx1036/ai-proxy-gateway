package collection

import (
	"fmt"
	"github.com/lx1036/gateway/pkg/config"
	"github.com/lx1036/gateway/pkg/config/schema/resource"
)

type Schemas struct {
	byCollection map[config.GroupVersionKind]resource.Schema
	byAddOrder   []resource.Schema
}

type SchemasBuilder struct {
	schemas Schemas
}

func NewSchemasBuilder() *SchemasBuilder {
	s := Schemas{
		byCollection: make(map[config.GroupVersionKind]resource.Schema),
	}

	return &SchemasBuilder{
		schemas: s,
	}
}

func (b *SchemasBuilder) Add(s resource.Schema) error {
	if _, found := b.schemas.byCollection[s.GroupVersionKind()]; found {
		return fmt.Errorf("collection already exists: %v", s.GroupVersionKind())
	}

	b.schemas.byCollection[s.GroupVersionKind()] = s
	b.schemas.byAddOrder = append(b.schemas.byAddOrder, s)
	return nil
}

// MustAdd calls Add and panics if it fails.
func (b *SchemasBuilder) MustAdd(s resource.Schema) *SchemasBuilder {
	if err := b.Add(s); err != nil {
		panic(fmt.Sprintf("SchemasBuilder.MustAdd: %v", err))
	}
	return b
}

func (b *SchemasBuilder) Build() Schemas {
	s := b.schemas

	// Avoid modify after Build.
	b.schemas = Schemas{}

	return s
}
