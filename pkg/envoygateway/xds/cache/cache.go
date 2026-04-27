package cache

import (
	"context"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"math"
	"strconv"

	"github.com/lx1036/gateway/pkg/envoygateway/xds/resource"
)

type SnapshotCache struct {
	snapshotVersion int64

	cachev3.SnapshotCache
}

func NewSnapshotCache(ads bool) *SnapshotCache {

	return &SnapshotCache{
		snapshotVersion: 0,

		SnapshotCache: cachev3.NewSnapshotCache(ads, &cachev3.IDHash{}, nil),
	}

}

func (snapshotCache *SnapshotCache) newSnapshotVersion() string {
	if snapshotCache.snapshotVersion == math.MaxInt64 {
		snapshotCache.snapshotVersion = 0
	}

	snapshotCache.snapshotVersion++

	return strconv.FormatInt(snapshotCache.snapshotVersion, 10)
}

func (snapshotCache *SnapshotCache) SetSnapshotCache(ctx context.Context, resources resource.XdsResources) {

	version := snapshotCache.newSnapshotVersion()
	resourceSnapshot, err := cachev3.NewSnapshot(version, resources)

	snapshotCache.SetSnapshot(ctx, node, resourceSnapshot)
}
