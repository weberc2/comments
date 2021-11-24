package objectstore

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

type GzipObjectStore struct {
	ObjectStore
}

func (os *GzipObjectStore) PutObject(bucket, key string, data io.ReadSeeker) error {
	var b bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("creating gzip writer: %w", err)
	}
	if _, err := io.Copy(w, data); err != nil {
		return fmt.Errorf("compressing data: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("closing gzip writer: %w", err)
	}
	return os.ObjectStore.PutObject(bucket, key, bytes.NewReader(b.Bytes()))
}

type GzipReadCloser struct {
	io.ReadCloser
	r *gzip.Reader
}

func (grc *GzipReadCloser) Read(data []byte) (int, error) {
	return grc.r.Read(data)
}

func (grc *GzipReadCloser) Close() error {
	if err := grc.ReadCloser.Close(); err != nil {
		return err
	}
	return grc.r.Close()
}

func (os *GzipObjectStore) GetObject(bucket, key string) (io.ReadCloser, error) {
	body, err := os.ObjectStore.GetObject(bucket, key)
	if err != nil {
		return nil, fmt.Errorf("getting object from storage: %w", err)
	}
	r, err := gzip.NewReader(body)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	return r, nil
}
