package framework

import (
	"github.com/lx1036/gateway/pkg/epp/datalayer"
	"github.com/lx1036/gateway/pkg/epp/datalayer/backend"
	"github.com/lx1036/gateway/pkg/epp/plugins"
	"sync"
)

type Pod interface {
	GetPod() *backend.Pod
	GetMetrics() *datalayer.Metrics
	String() string
	//Get(string) (datalayer.Cloneable, bool)
	//Put(string, datalayer.Cloneable)
	Keys() []string
}

type ProfileRunResult struct {
	TargetPods []Pod
}


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
