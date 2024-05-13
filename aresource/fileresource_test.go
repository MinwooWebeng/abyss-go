package aresource

import "testing"

func TestFileResourceCreation(t *testing.T) {
	lazy_file, err := MakeLazyFileResource("C:\\Users\\minwoo\\Desktop\\goghcut.png", "gogh_paint")
	if err != nil {
		t.Fatalf("failed to make lazy file resource: %s\n", err.Error())
	}
	if lazy_file.Path() != "gogh_paint" {
		t.Fatalf("MakeLazyFileResource: Path() not match")
	}
	buf := make([]byte, 1024)
	if lazy_file.Read(0, buf) == 0 {
		t.Fatalf("file Read failed")
	}
}
