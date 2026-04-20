package kvblock

import "fmt"

type KVCacheIndexer struct {
}

func NewKVCacheIndexer(tokenProcessor *ChunkedTokenDatabase) (*KVCacheIndexer, error) {
	if config == nil {
		config = NewDefaultConfig()
	}
	if tokenProcessor == nil {
		return nil, fmt.Errorf("tokenProcessor cannot be nil")
	}

}
