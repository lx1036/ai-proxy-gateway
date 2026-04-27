package cache

import (
	"context"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/lx1036/gateway/pkg/envoygateway/xds/resource"
	"math"
	"strconv"
)

type SnapshotCacheWithCallbacks interface {
	cachev3.SnapshotCache
	serverv3.Callbacks

	SetSnapshotCache(ctx context.Context, node string, resources resource.XdsResources) error
}


type SnapshotCache struct {
	snapshotVersion int64

	cachev3.SnapshotCache
}

func NewSnapshotCache(ads bool) SnapshotCacheWithCallbacks {

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

func (snapshotCache *SnapshotCache) SetSnapshotCache(ctx context.Context, node string, resources resource.XdsResources) error {
	version := snapshotCache.newSnapshotVersion()
	resourceSnapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		return err
	}

	return snapshotCache.SetSnapshot(ctx, node, resourceSnapshot)
}


func (snapshotCache *SnapshotCache) OnFetchRequest(ctx context.Context, request *discovery.DiscoveryRequest) error {
	//TODO implement me
	panic("implement me")
}

func (snapshotCache *SnapshotCache) OnFetchResponse(request *discovery.DiscoveryRequest, response *discovery.DiscoveryResponse) {
	//TODO implement me
	panic("implement me")
}

func (snapshotCache *SnapshotCache) OnStreamOpen(ctx context.Context, streamID int64, s string) error {
	//TODO implement me
	panic("implement me")
}

func (snapshotCache *SnapshotCache) OnStreamClosed(streamID int64, node *corev3.Node) {
	//TODO implement me
	panic("implement me")
}

func (snapshotCache *SnapshotCache) OnStreamRequest(i int64, request *discovery.DiscoveryRequest) error {
	//TODO implement me
	panic("implement me")
}

func (snapshotCache *SnapshotCache) OnStreamResponse(ctx context.Context, i int64, request *discovery.DiscoveryRequest, response *discovery.DiscoveryResponse) {
	//TODO implement me
	panic("implement me")
}

func (snapshotCache *SnapshotCache) OnDeltaStreamOpen(ctx context.Context, i int64, s string) error {
	//TODO implement me
	panic("implement me")
}

func (snapshotCache *SnapshotCache) OnDeltaStreamClosed(i int64, node *corev3.Node) {
	//TODO implement me
	panic("implement me")
}

func (snapshotCache *SnapshotCache) OnStreamDeltaRequest(i int64, request *discovery.DeltaDiscoveryRequest) error {
	//TODO implement me
	panic("implement me")
}

func (snapshotCache *SnapshotCache) OnStreamDeltaResponse(i int64, request *discovery.DeltaDiscoveryRequest, response *discovery.DeltaDiscoveryResponse) {
	//TODO implement me
	panic("implement me")
}

