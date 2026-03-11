package xds

func (s *DiscoveryServer) DeltaAggregatedResources(stream DeltaDiscoveryStream) error {
	return s.StreamDeltas(stream)
}

func (s *DiscoveryServer) StreamDeltas(stream DeltaDiscoveryStream) error {

}
