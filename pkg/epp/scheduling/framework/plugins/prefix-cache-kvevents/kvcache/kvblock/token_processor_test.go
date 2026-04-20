package kvblock

import (
	"hash/fnv"
	"k8s.io/klog/v2"
	"testing"
)

func TestTokenProcessor(test *testing.T) {
	hashSeed := ""
	h := fnv.New64a()
	_, _ = h.Write([]byte(hashSeed))
	initHash := h.Sum64()
	klog.Infof("initHash: %d", initHash) // 14695981039346656037
}
