package aresource

type ResourceMetadata struct {
	path string
}

func (m ResourceMetadata) Path() string {
	return m.path
}
