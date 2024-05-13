package aresource

type IFileResource interface {
	IAbyssResource

	Read(file_offset int64, buf []byte) int
}
