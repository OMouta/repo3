package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrNoSuchBucket      = errors.New("no such bucket")
	ErrNoSuchKey         = errors.New("no such key")
	ErrBucketExists      = errors.New("bucket already exists")
	ErrBucketNotEmpty    = errors.New("bucket not empty")
	ErrInvalidBucketName = errors.New("invalid bucket name")
	ErrInvalidKey        = errors.New("invalid key")
	ErrAccessDenied      = errors.New("access denied")
	ErrSlowDown          = errors.New("slow down")
	ErrOperationAborted  = errors.New("operation aborted")
	ErrValidation        = errors.New("validation failed")
)

type Bucket struct {
	Name         string
	CreationDate time.Time
}

type Metadata struct {
	ContentType  string
	UserMetadata map[string]string
}

type PutResult struct {
	ETag      string
	VersionID string
}

type Object struct {
	Info ObjectInfo
	Body io.ReadCloser
}

type ObjectInfo struct {
	Bucket       string
	Key          string
	Size         int64
	ETag         string
	LastModified time.Time
	ContentType  string
	VersionID    string
	UserMetadata map[string]string
}

type ObjectStore interface {
	ListBuckets(ctx context.Context) ([]Bucket, error)
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error

	PutObject(ctx context.Context, bucket, key string, body io.Reader, meta Metadata) (*PutResult, error)
	GetObject(ctx context.Context, bucket, key string, versionID string) (*Object, error)
	HeadObject(ctx context.Context, bucket, key string, versionID string) (*ObjectInfo, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	ListObjects(ctx context.Context, bucket, prefix string, limit int) ([]ObjectInfo, error)
}
