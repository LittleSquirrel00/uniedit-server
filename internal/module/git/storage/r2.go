package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// R2Config holds R2 storage configuration.
type R2Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

// R2Client wraps the S3 client for R2 operations.
type R2Client struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

// NewR2Client creates a new R2 client.
func NewR2Client(cfg *R2Config) (*R2Client, error) {
	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.Bucket == "" {
		return nil, errors.New("incomplete R2 configuration")
	}

	// Create credentials provider
	creds := credentials.NewStaticCredentialsProvider(
		cfg.AccessKeyID,
		cfg.SecretAccessKey,
		"",
	)

	// Set region (R2 uses "auto" but we need a valid region for SDK)
	region := cfg.Region
	if region == "" {
		region = "auto"
	}

	// Load AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(creds),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	// Create S3 client with R2 endpoint
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true // R2 requires path-style URLs
	})

	return &R2Client{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    cfg.Bucket,
	}, nil
}

// PresignedURL represents a presigned URL response.
type PresignedURL struct {
	URL       string
	Method    string
	ExpiresAt time.Time
}

// PresignUpload generates a presigned URL for uploading an object.
func (c *R2Client) PresignUpload(ctx context.Context, key string, size int64, expiry time.Duration) (*PresignedURL, error) {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		ContentLength: aws.Int64(size),
	}

	req, err := c.presigner.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return nil, fmt.Errorf("presign put: %w", err)
	}

	return &PresignedURL{
		URL:       req.URL,
		Method:    req.Method,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

// PresignDownload generates a presigned URL for downloading an object.
func (c *R2Client) PresignDownload(ctx context.Context, key string, expiry time.Duration) (*PresignedURL, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	req, err := c.presigner.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return nil, fmt.Errorf("presign get: %w", err)
	}

	return &PresignedURL{
		URL:       req.URL,
		Method:    req.Method,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

// PutObject uploads an object to R2.
func (c *R2Client) PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	_, err := c.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}

// GetObject retrieves an object from R2.
func (c *R2Client) GetObject(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	result, err := c.client.GetObject(ctx, input)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, 0, ErrObjectNotFound
		}
		return nil, 0, fmt.Errorf("get object: %w", err)
	}

	size := int64(0)
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	return result.Body, size, nil
}

// HeadObject checks if an object exists and returns its metadata.
func (c *R2Client) HeadObject(ctx context.Context, key string) (*ObjectInfo, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	result, err := c.client.HeadObject(ctx, input)
	if err != nil {
		var nsk *types.NoSuchKey
		var nf *types.NotFound
		if errors.As(err, &nsk) || errors.As(err, &nf) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("head object: %w", err)
	}

	size := int64(0)
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	contentType := ""
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	return &ObjectInfo{
		Key:          key,
		Size:         size,
		ContentType:  contentType,
		LastModified: result.LastModified,
	}, nil
}

// ObjectExists checks if an object exists.
func (c *R2Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := c.HeadObject(ctx, key)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteObject deletes an object from R2.
func (c *R2Client) DeleteObject(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	return nil
}

// DeleteObjects deletes multiple objects from R2.
func (c *R2Client) DeleteObjects(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = types.ObjectIdentifier{
			Key: aws.String(key),
		}
	}

	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(c.bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	}

	_, err := c.client.DeleteObjects(ctx, input)
	if err != nil {
		return fmt.Errorf("delete objects: %w", err)
	}

	return nil
}

// ListObjects lists objects with a given prefix.
func (c *R2Client) ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]*ObjectInfo, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(c.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(maxKeys),
	}

	result, err := c.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}

	objects := make([]*ObjectInfo, 0, len(result.Contents))
	for _, obj := range result.Contents {
		info := &ObjectInfo{
			LastModified: obj.LastModified,
		}
		if obj.Key != nil {
			info.Key = *obj.Key
		}
		if obj.Size != nil {
			info.Size = *obj.Size
		}
		objects = append(objects, info)
	}

	return objects, nil
}

// CopyObject copies an object within R2.
func (c *R2Client) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(c.bucket),
		CopySource: aws.String(c.bucket + "/" + srcKey),
		Key:        aws.String(dstKey),
	}

	_, err := c.client.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("copy object: %w", err)
	}

	return nil
}

// ObjectInfo represents object metadata.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified *time.Time
}

// Storage errors.
var (
	ErrObjectNotFound = errors.New("object not found")
)
