package aresource

type IResourceAccessBase interface {
	IsAccessibleFrom(access_group AccessGroup) bool
	SetAccessGroups(access_groups []AccessGroup)
	AddAccessGroup(access_group AccessGroup)
	RemoveAccessGroup(access_group AccessGroup)
}
