package objectstore

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/weberc2/comments/pkg/types"
)

type S3ObjectStore struct {
	Client *s3.S3
}

func (os *S3ObjectStore) PutObject(bucket, key string, data io.ReadSeeker) error {
	if _, err := os.Client.PutObject(&s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   data,
	}); err != nil {
		return fmt.Errorf(
			"putting object in bucket `%s` at key `%s`: %w",
			bucket,
			key,
			err,
		)
	}
	return nil
}

func (os *S3ObjectStore) GetObject(bucket, key string) (io.ReadCloser, error) {
	rsp, err := os.Client.GetObject(&s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() == s3.ErrCodeNoSuchKey {
				return nil, &types.ObjectNotFoundErr{Bucket: bucket, Key: key}
			}
		}
		return nil, fmt.Errorf(
			"getting object from bucket `%s` at key `%s`: %w",
			bucket,
			key,
			err,
		)
	}
	return rsp.Body, nil
}

func (os *S3ObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	var keys []string
	if err := os.Client.ListObjectsPages(
		&s3.ListObjectsInput{
			Bucket: &bucket,
			Prefix: &prefix,
		},
		func(rsp *s3.ListObjectsOutput, lastPage bool) bool {
			for _, object := range rsp.Contents {
				keys = append(keys, *object.Key)
			}
			return true
		},
	); err != nil {
		return keys, fmt.Errorf(
			"listing objects in bucket `%s` with prefix `%s`: %w",
			bucket,
			prefix,
			err,
		)
	}
	return keys, nil
}

func (os *S3ObjectStore) DeleteObject(bucket, key string) error {
	if _, err := os.Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}); err != nil {
		return fmt.Errorf("deleting objects: %w", err)
	}
	return nil
}
