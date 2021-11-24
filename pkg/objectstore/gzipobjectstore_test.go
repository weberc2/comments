package objectstore

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/weberc2/comments/pkg/testsupport"
)

func TestGzipObjectStore(t *testing.T) {
	objectStore := GzipObjectStore{testsupport.ObjectStoreFake{}}
	if err := objectStore.PutObject(
		"my-bucket",
		"my-key",
		strings.NewReader("my-data"),
	); err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}

	body, err := objectStore.GetObject("my-bucket", "my-key")
	if err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
	if string(data) != "my-data" {
		t.Fatalf("wanted 'my-data'; found '%s'", data)
	}
}
