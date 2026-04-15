package exportmanager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type s3Store struct {
	client *s3.Client
}

func NewS3Store(client *s3.Client) ObjectStore {
	return &s3Store{client: client}
}

func (s *s3Store) EnsureBucket(ctx context.Context, bucket string) error {
	if bucket == "" {
		return fmt.Errorf("bucket invalido")
	}
	if !shouldAutoCreateBucket() {
		return nil
	}
	if _, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err == nil {
		return nil
	}
	_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorCode() == "BucketAlreadyOwnedByYou" || apiErr.ErrorCode() == "BucketAlreadyExists" {
			return nil
		}
	}
	return err
}

func (s *s3Store) PutObject(ctx context.Context, bucket string, key string, body []byte, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	}
	_, err := s.client.PutObject(ctx, input)
	if err == nil {
		return nil
	}
	if isNoSuchBucket(err) && shouldAutoCreateBucket() {
		if ensureErr := s.EnsureBucket(ctx, bucket); ensureErr != nil {
			return fmt.Errorf("put object failed with missing bucket and ensure bucket also failed: %w", ensureErr)
		}
		_, retryErr := s.client.PutObject(ctx, input)
		return retryErr
	}
	return err
}

func (s *s3Store) GetObject(ctx context.Context, bucket string, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *s3Store) HeadObject(ctx context.Context, bucket string, key string) (ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{ContentLength: aws.ToInt64(out.ContentLength)}, nil
}

func (s *s3Store) DeleteObject(ctx context.Context, bucket string, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *s3Store) CreateMultipartUpload(ctx context.Context, bucket string, key string, contentType string) (string, error) {
	out, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.UploadId), nil
}

func (s *s3Store) UploadPart(ctx context.Context, bucket string, key string, uploadID string, partNumber int32, body []byte) (string, error) {
	out, err := s.client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		UploadId:      aws.String(uploadID),
		PartNumber:    aws.Int32(partNumber),
		Body:          bytes.NewReader(body),
		ContentLength: aws.Int64(int64(len(body))),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.ETag), nil
}

func (s *s3Store) CompleteMultipartUpload(ctx context.Context, bucket string, key string, uploadID string, parts []CompletedPart) error {
	completed := make([]s3types.CompletedPart, 0, len(parts))
	for _, p := range parts {
		etag := p.ETag
		completed = append(completed, s3types.CompletedPart{
			ETag:       aws.String(etag),
			PartNumber: aws.Int32(p.PartNumber),
		})
	}
	_, err := s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3types.CompletedMultipartUpload{
			Parts: completed,
		},
	})
	return err
}

func (s *s3Store) AbortMultipartUpload(ctx context.Context, bucket string, key string, uploadID string) error {
	_, err := s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	return err
}

func shouldAutoCreateBucket() bool {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "local" || appEnv == "" {
		return true
	}
	endpoint := strings.ToLower(os.Getenv("AWS_ENDPOINT_URL"))
	localstack := strings.ToLower(os.Getenv("LOCALSTACK_ENDPOINT_BASE"))
	return strings.Contains(endpoint, "localhost") ||
		strings.Contains(endpoint, "localstack") ||
		strings.Contains(localstack, "localhost") ||
		strings.Contains(localstack, "localstack")
}

func isNoSuchBucket(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NoSuchBucket" || apiErr.ErrorCode() == "NotFound"
	}
	return strings.Contains(err.Error(), "NoSuchBucket")
}
