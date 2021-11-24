package testsupport

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	"github.com/weberc2/comments/pkg/types"
)

type ObjectStoreFake map[[2]string][]byte

func (osf ObjectStoreFake) PutObject(
	bucket string,
	key string,
	data io.ReadSeeker,
) error {
	var b bytes.Buffer
	if _, err := io.Copy(&b, data); err != nil {
		return err
	}
	osf[[2]string{bucket, key}] = b.Bytes()
	return nil
}

func (osf ObjectStoreFake) GetObject(
	bucket string,
	key string,
) (io.ReadCloser, error) {
	data, found := osf[[2]string{bucket, key}]
	if !found {
		return nil, &types.ObjectNotFoundErr{Bucket: bucket, Key: key}
	}
	return ioutil.NopCloser(bytes.NewReader(data)), nil
}

func (osf ObjectStoreFake) ListObjects(
	bucket string,
	prefix string,
) ([]string, error) {
	var out []string
	for key := range osf {
		if key[0] == bucket && strings.HasPrefix(key[1], prefix) {
			out = append(out, key[1])
		}
	}
	return out, nil
}

func (osf ObjectStoreFake) DeleteObject(bucket, key string) error {
	k := [2]string{bucket, key}
	if _, found := osf[k]; !found {
		return &types.ObjectNotFoundErr{Bucket: bucket, Key: key}
	}
	delete(osf, k)
	return nil
}
