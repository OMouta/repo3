package client

import (
	"fmt"
	"strings"
)

type S3Path struct {
	Bucket string
	Key    string
}

func ParseS3Path(raw string) (S3Path, error) {
	if !strings.HasPrefix(raw, "s3://") {
		return S3Path{}, fmt.Errorf("invalid S3 path %q: must start with s3://", raw)
	}

	rest := strings.TrimPrefix(raw, "s3://")
	if rest == "" || strings.HasPrefix(rest, "/") {
		return S3Path{}, fmt.Errorf("invalid S3 path %q", raw)
	}

	bucket, key, _ := strings.Cut(rest, "/")
	if bucket == "" {
		return S3Path{}, fmt.Errorf("invalid S3 path %q: missing bucket", raw)
	}
	return S3Path{Bucket: bucket, Key: key}, nil
}
