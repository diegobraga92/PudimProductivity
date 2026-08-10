package media

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Uploader issues presigned PUT URLs against an S3 bucket. It reads
// credentials from the standard AWS credential chain (env, shared config, IAM
// role) — nothing is stored in the repo.
type S3Uploader struct {
	presigner *s3.PresignClient
	bucket    string
}

// NewS3Uploader builds an uploader. bucket and region are required; the
// credentials come from the environment (AWS_ACCESS_KEY_ID / role).
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
		bucket:    bucket,
	}, nil
}

// GenerateUploadURL returns a presigned PUT URL valid for ttl.
func (s *S3Uploader) GenerateUploadURL(ctx context.Context, key, contentType string, ttl time.Duration) (*UploadURL, error) {
	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return nil, fmt.Errorf("media: presign upload URL: %w", err)
	}
	return &UploadURL{URL: req.URL, Key: key}, nil
}
