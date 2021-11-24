package objectstore

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

func TestGzipObjectStore(t *testing.T) {
	objectStore := GzipObjectStore{objectStoreFake{}}
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
