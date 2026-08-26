package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/media"
)

// S3Uploader issues presigned PUT URLs against an S3 bucket and implements the
// media.Storage port for direct object reads/writes. It reads credentials from
// the standard AWS credential chain (env, shared config, IAM role).
type S3Uploader struct {
	presigner *s3.PresignClient
	client    *s3.Client
	bucket    string
}

// Compile-time check that S3Uploader satisfies the media.Storage port.
var _ media.Storage = (*S3Uploader)(nil)

// NewS3Uploader builds an uploader. bucket and region are required.
// The credentials come from the environment (AWS_ACCESS_KEY_ID / role).
func NewS3Uploader(ctx context.Context, bucket, region string) (*S3Uploader, error) {
	if bucket == "" || region == "" {
		return nil, fmt.Errorf("media: S3 bucket and region are required")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("media: load AWS config: %w", err)
	}
	client := s3.NewFromConfig(cfg)
	return &S3Uploader{
		presigner: s3.NewPresignClient(client),
		client:    client,
		bucket:    bucket,
	}, nil
}

// GenerateUploadURL returns a presigned PUT URL valid for ttl.
func (s *S3Uploader) GenerateUploadURL(ctx context.Context, key, contentType string, ttl time.Duration) (*media.UploadURL, error) {
	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return nil, fmt.Errorf("media: presign upload URL: %w", err)
	}
	return &media.UploadURL{URL: req.URL, Key: key}, nil
}

// Get implements media.Storage.
func (s *S3Uploader) Get(ctx context.Context, key string) (*media.File, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nf *types.NoSuchKey
		if errors.As(err, &nf) {
			return nil, media.ErrNotFound
		}
		return nil, fmt.Errorf("media: get object %q: %w", key, err)
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return &media.File{Key: key, Content: out.Body, Size: size}, nil
}

// Put implements media.Storage.
func (s *S3Uploader) Put(ctx context.Context, key, contentType string, r io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("media: put object %q: %w", key, err)
	}
	return nil
}

// Delete implements media.Storage.
func (s *S3Uploader) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("media: delete object %q: %w", key, err)
	}
	return nil
}
