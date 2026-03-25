package types

import (
	"sync"

	"github.com/lx1036/gateway/pkg/epp/plugins"
)

type CycleState struct {
	// key: StateKey, value: StateData
	storage sync.Map
}

func NewCycleState() *CycleState {
	return &CycleState{}
}

func (c *CycleState) Read(key plugins.StateKey) (plugins.StateData, error) {
	if v, ok := c.storage.Load(key); ok {
		return v.(plugins.StateData), nil
	}

	return nil, plugins.ErrNotFound
}

func (c *CycleState) Write(key plugins.StateKey, val plugins.StateData) {
	c.storage.Store(key, val)
}

func (c *CycleState) Delete(key plugins.StateKey) {
	c.storage.Delete(key)
}
