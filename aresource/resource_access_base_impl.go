package aresource

type ResourceAccessBase struct {
	_access_map *AccessMap
}

type AccessMap struct {
	_inner map[AccessGroup]bool
}

func MakeResourceAccessBase() ResourceAccessBase {
	access_base := ResourceAccessBase{new(AccessMap)}
	access_base._access_map._inner = make(map[AccessGroup]bool)
	return access_base
}

func (access ResourceAccessBase) IsAccessibleFrom(access_group AccessGroup) bool {
	_, exist := access._access_map._inner[access_group]
	return exist
}
func (access ResourceAccessBase) SetAccessGroups(access_groups []AccessGroup) {
	access._access_map._inner = make(map[AccessGroup]bool)
	for _, access_group := range access_groups {
		access.AddAccessGroup(access_group)
	}
}
func (access ResourceAccessBase) AddAccessGroup(access_group AccessGroup) {
	access._access_map._inner[access_group] = true
}
func (access ResourceAccessBase) RemoveAccessGroup(access_group AccessGroup) {
	delete(access._access_map._inner, access_group)
}
