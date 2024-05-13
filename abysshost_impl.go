package abyssgo

import "context"

type AbyssHost struct {
	credential *AbyssHostCredential

	remote_resource_provider   IRemoteResourceProvider
	neighbor_discovery_handler INeighborDiscoveryHandler
	resource_access_authorizer IResourceAccessAuthorizer
	realtime_resource_handler  IRealtimeResourceHandler
}

func (host AbyssHost) RunNetworkService(ctx context.Context) {

}
func (host AbyssHost) TerminateNetworkService() {

}

func (host AbyssHost) GetIRemoteResourceProvider() IRemoteResourceProvider {
	return host.remote_resource_provider
}
func (host AbyssHost) GetINeighborDiscoveryHandler() INeighborDiscoveryHandler {
	return host.neighbor_discovery_handler
}
func (host AbyssHost) GetIResourceAccessAuthorizer() IResourceAccessAuthorizer {
	return host.resource_access_authorizer
}
func (host AbyssHost) GetIRealtimeResourceHandler() IRealtimeResourceHandler {
	return host.realtime_resource_handler
}

func MakeAbysshost(credential *AbyssHostCredential) (IAbyssHost, error) {
	result := AbyssHost{}
	return result, nil
}
