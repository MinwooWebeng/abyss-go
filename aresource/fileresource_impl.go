package aresource

import (
	"io"
	"os"
)

type LazyFileResource struct {
	ResourceAccessBase
	ResourceMetadata

	_file        *OSFileBase
	_TryOpenFile func() bool
}

type OSFileBase struct {
	_os_file *os.File
}

func (f LazyFileResource) Read(file_offset int64, buf []byte) int {
	if f._file._os_file == nil {
		if !f._TryOpenFile() {
			return 0
		}
	}
	n, err := f._file._os_file.ReadAt(buf, file_offset)
	if err != nil && err != io.EOF {
		return 0
	}
	return n
}

func MakeLazyFileResource(filepath string, querypath string) (IFileResource, error) {
	file_resource := LazyFileResource{
		MakeResourceAccessBase(),
		ResourceMetadata{querypath},
		new(OSFileBase),
		nil,
	}
	file_resource._TryOpenFile = func() bool {
		//TODO: load metadata
		os_file, err := os.Open(filepath)
		if err != nil {
			return false
		}
		file_resource._file._os_file = os_file
		return true
	}
	//TODO: elaborate metadata
	return file_resource, nil
}
