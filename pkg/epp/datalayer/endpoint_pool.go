package datalayer

type EndpointPool struct {
	Selector    map[string]string
	TargetPorts []int
	Namespace   string
	Name        string
}

type EndpointLifecycle struct {
}

//func (lc *EndpointLifecycle) NewEndpoint(parent context.Context, inEndpointMetadata *EndpointMetadata, _ PoolInfo) Endpoint {
//
//}
