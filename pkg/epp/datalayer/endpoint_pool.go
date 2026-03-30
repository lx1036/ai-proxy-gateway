package datalayer

type EndpointPool struct {
	Namespace   string
	Name        string
	Selector    map[string]string
	TargetPorts []int
}

type EndpointLifecycle struct {
}

//func (lc *EndpointLifecycle) NewEndpoint(parent context.Context, inEndpointMetadata *EndpointMetadata, _ PoolInfo) Endpoint {
//
//}
