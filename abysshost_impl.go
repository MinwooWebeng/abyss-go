package abyssgo

import (
	"abyss/and"
	"context"
)

type AbyssHost struct {
	credential *AbyssHostCredential

	remote_resource_provider   IRemoteResourceProvider
	neighbor_discovery_handler and.INeighborDiscoveryHandler
	resource_access_authorizer IResourceAccessAuthorizer
	realtime_resource_handler  IRealtimeResourceHandler
}

func MakeAbysshost(credential *AbyssHostCredential) (IAbyssHost, error) {
	var result AbyssHost
	result.credential = credential
	result.remote_resource_provider = NewRemoteResourceProvider()
	result.neighbor_discovery_handler = and.NewNeighborDiscoveryHandler()

	return result, nil
}

func (host AbyssHost) RunNetworkService(ctx context.Context) {

}
func (host AbyssHost) TerminateNetworkService() {
}

func (host AbyssHost) GetIRemoteResourceProvider() IRemoteResourceProvider {
	return host.remote_resource_provider
}
func (host AbyssHost) GetINeighborDiscoveryHandler() and.INeighborDiscoveryHandler {
	return host.neighbor_discovery_handler
}
func (host AbyssHost) GetIResourceAccessAuthorizer() IResourceAccessAuthorizer {
	return host.resource_access_authorizer
}
func (host AbyssHost) GetIRealtimeResourceHandler() IRealtimeResourceHandler {
	return host.realtime_resource_handler
}
