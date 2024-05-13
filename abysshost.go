package abyssgo

import (
	"context"
)

type IAbyssHost interface {
	RunNetworkService(ctx context.Context)
	TerminateNetworkService()

	GetIRemoteResourceProvider() IRemoteResourceProvider
	GetINeighborDiscoveryHandler() INeighborDiscoveryHandler
	GetIResourceAccessAuthorizer() IResourceAccessAuthorizer
	GetIRealtimeResourceHandler() IRealtimeResourceHandler
}
