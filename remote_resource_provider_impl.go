package abyssgo

type RemoteResourceProvider struct {
}

func NewRemoteResourceProvider() IRemoteResourceProvider {
	return RemoteResourceProvider{}
}
