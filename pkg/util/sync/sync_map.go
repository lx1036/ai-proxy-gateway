package sync

import (
	"sync"
	"sync/atomic"
)

// 泛型 sync.Map
type SyncMap[K any, V any] struct {
	m   sync.Map
	len atomic.Int32
}
