package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/uniedit/server/internal/port/outbound"
)

// GitLFSStorageAdapter implements GitLFSStoragePort using R2/S3.
type GitLFSStorageAdapter struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	prefix    string
}

// NewGitLFSStorageAdapter creates a new Git LFS storage adapter.
func NewGitLFSStorageAdapter(client *s3.Client, bucket, prefix string) *GitLFSStorageAdapter {
	return &GitLFSStorageAdapter{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    bucket,
		prefix:    prefix,
	}
}

func (a *GitLFSStorageAdapter) key(oid string) string {
	return a.prefix + oid
}

// Upload uploads an LFS object.
func (a *GitLFSStorageAdapter) Upload(ctx context.Context, oid string, reader io.Reader, size int64) error {
	_, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(a.bucket),
		Key:           aws.String(a.key(oid)),
		Body:          reader,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String("application/octet-stream"),
	})
	if err != nil {
		return fmt.Errorf("upload LFS object: %w", err)
	}

	return nil
}

// Download downloads an LFS object.
func (a *GitLFSStorageAdapter) Download(ctx context.Context, oid string) (io.ReadCloser, int64, error) {
	result, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(a.key(oid)),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, 0, ErrObjectNotFound
		}
		return nil, 0, fmt.Errorf("download LFS object: %w", err)
	}

	size := int64(0)
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	return result.Body, size, nil
}

// Exists checks if an LFS object exists.
func (a *GitLFSStorageAdapter) Exists(ctx context.Context, oid string) (bool, error) {
	_, err := a.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(a.key(oid)),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		var nf *types.NotFound
		if errors.As(err, &nsk) || errors.As(err, &nf) {
			return false, nil
		}
		return false, fmt.Errorf("check LFS object: %w", err)
	}

	return true, nil
}

// Delete deletes an LFS object.
func (a *GitLFSStorageAdapter) Delete(ctx context.Context, oid string) error {
	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(a.key(oid)),
	})
	if err != nil {
		return fmt.Errorf("delete LFS object: %w", err)
	}

	return nil
}

// GenerateUploadURL generates a presigned upload URL.
func (a *GitLFSStorageAdapter) GenerateUploadURL(ctx context.Context, oid string, size int64, expiry time.Duration) (*outbound.GitPresignedURL, error) {
	req, err := a.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(a.bucket),
		Key:           aws.String(a.key(oid)),
		ContentLength: aws.Int64(size),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return nil, fmt.Errorf("presign upload: %w", err)
	}

	return &outbound.GitPresignedURL{
		URL:       req.URL,
		Method:    req.Method,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

// GenerateDownloadURL generates a presigned download URL.
func (a *GitLFSStorageAdapter) GenerateDownloadURL(ctx context.Context, oid string, expiry time.Duration) (*outbound.GitPresignedURL, error) {
	req, err := a.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(a.key(oid)),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return nil, fmt.Errorf("presign download: %w", err)
	}

	return &outbound.GitPresignedURL{
		URL:       req.URL,
		Method:    req.Method,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

// Compile-time check
var _ outbound.GitLFSStoragePort = (*GitLFSStorageAdapter)(nil)
