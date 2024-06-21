package abyssgo

import (
	"abyss/and"
	"context"
)

type IAbyssHost interface {
	RunNetworkService(ctx context.Context)
	TerminateNetworkService()

	GetIRemoteResourceProvider() IRemoteResourceProvider
	GetINeighborDiscoveryHandler() and.INeighborDiscoveryHandler
	GetIResourceAccessAuthorizer() IResourceAccessAuthorizer
	GetIRealtimeResourceHandler() IRealtimeResourceHandler
}
