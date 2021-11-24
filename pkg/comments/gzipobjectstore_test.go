package comments

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

type objectStoreFake map[[2]string][]byte

func (osf objectStoreFake) PutObject(bucket, key string, data io.ReadSeeker) error {
	var b bytes.Buffer
	if _, err := io.Copy(&b, data); err != nil {
		return err
	}
	osf[[2]string{bucket, key}] = b.Bytes()
	return nil
}

func (osf objectStoreFake) GetObject(bucket, key string) (io.ReadCloser, error) {
	data, found := osf[[2]string{bucket, key}]
	if !found {
		return nil, &ObjectNotFoundErr{Bucket: bucket, Key: key}
	}
	return ioutil.NopCloser(bytes.NewReader(data)), nil
}

func (osf objectStoreFake) ListObjects(bucket, prefix string) ([]string, error) {
	var out []string
	for key := range osf {
		if key[0] == bucket && strings.HasPrefix(key[1], prefix) {
			out = append(out, key[1])
		}
	}
	return out, nil
}

func (osf objectStoreFake) DeleteObject(bucket, key string) error {
	k := [2]string{bucket, key}
	if _, found := osf[k]; !found {
		return &ObjectNotFoundErr{Bucket: bucket, Key: key}
	}
	delete(osf, k)
	return nil
}

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
