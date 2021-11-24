package types

import (
	"fmt"
	"io"
)

type ObjectNotFoundErr struct {
	Bucket string
	Key    string
}

func (err *ObjectNotFoundErr) Error() string {
	return fmt.Sprintf(
		"object not found (bucket=%s) (key=%s)",
		err.Bucket,
		err.Key,
	)
}

type ObjectStore interface {
	PutObject(bucket, key string, data io.ReadSeeker) error
	GetObject(bucket, key string) (io.ReadCloser, error)
	ListObjects(bucket, prefix string) ([]string, error)
	DeleteObject(bucket, key string) error
}
