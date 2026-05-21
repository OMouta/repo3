package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryStore struct {
	mu      sync.RWMutex
	buckets map[string]*memoryBucket
}

type memoryBucket struct {
	created time.Time
	objects map[string]memoryObject
}

type memoryObject struct {
	data []byte
	info ObjectInfo
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{buckets: make(map[string]*memoryBucket)}
}

func (s *MemoryStore) ListBuckets(ctx context.Context) ([]Bucket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	buckets := make([]Bucket, 0, len(s.buckets))
	for name, bucket := range s.buckets {
		buckets = append(buckets, Bucket{Name: name, CreationDate: bucket.created})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Name < buckets[j].Name })
	return buckets, nil
}

func (s *MemoryStore) CreateBucket(ctx context.Context, name string) error {
	if !validBucketName(name) {
		return ErrInvalidBucketName
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.buckets[name]; ok {
		return ErrBucketExists
	}
	s.buckets[name] = &memoryBucket{created: time.Now().UTC(), objects: make(map[string]memoryObject)}
	return nil
}

func (s *MemoryStore) DeleteBucket(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, ok := s.buckets[name]
	if !ok {
		return ErrNoSuchBucket
	}
	if len(bucket.objects) > 0 {
		return ErrBucketNotEmpty
	}
	delete(s.buckets, name)
	return nil
}

func (s *MemoryStore) PutObject(ctx context.Context, bucket, key string, body io.Reader, meta Metadata) (*PutResult, error) {
	if !validKey(key) {
		return nil, ErrInvalidKey
	}

	data, err := io.ReadAll(io.LimitReader(body, 10*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 10*1024*1024 {
		return nil, fmt.Errorf("object too large")
	}

	now := time.Now().UTC()
	etag := md5Hex(data)

	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrNoSuchBucket
	}
	versionID := fmt.Sprintf("%d", now.UnixNano())
	b.objects[key] = memoryObject{
		data: append([]byte(nil), data...),
		info: ObjectInfo{
			Bucket:       bucket,
			Key:          key,
			Size:         int64(len(data)),
			ETag:         etag,
			LastModified: now,
			ContentType:  meta.ContentType,
			VersionID:    versionID,
		},
	}
	return &PutResult{ETag: etag, VersionID: versionID}, nil
}

func (s *MemoryStore) GetObject(ctx context.Context, bucket, key string, versionID string) (*Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, err := s.getObjectLocked(bucket, key)
	if err != nil {
		return nil, err
	}
	return &Object{
		Info: obj.info,
		Body: io.NopCloser(bytes.NewReader(append([]byte(nil), obj.data...))),
	}, nil
}

func (s *MemoryStore) HeadObject(ctx context.Context, bucket, key string, versionID string) (*ObjectInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, err := s.getObjectLocked(bucket, key)
	if err != nil {
		return nil, err
	}
	info := obj.info
	return &info, nil
}

func (s *MemoryStore) DeleteObject(ctx context.Context, bucket, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[bucket]
	if !ok {
		return ErrNoSuchBucket
	}
	if _, ok := b.objects[key]; !ok {
		return ErrNoSuchKey
	}
	delete(b.objects, key)
	return nil
}

func (s *MemoryStore) ListObjects(ctx context.Context, bucket, prefix string, limit int) ([]ObjectInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrNoSuchBucket
	}

	keys := make([]string, 0, len(b.objects))
	for key := range b.objects {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	if limit <= 0 || limit > len(keys) {
		limit = len(keys)
	}
	out := make([]ObjectInfo, 0, limit)
	for _, key := range keys[:limit] {
		out = append(out, b.objects[key].info)
	}
	return out, nil
}

func (s *MemoryStore) getObjectLocked(bucket, key string) (memoryObject, error) {
	b, ok := s.buckets[bucket]
	if !ok {
		return memoryObject{}, ErrNoSuchBucket
	}
	obj, ok := b.objects[key]
	if !ok {
		return memoryObject{}, ErrNoSuchKey
	}
	return obj, nil
}

func validBucketName(name string) bool {
	if len(name) < 3 || len(name) > 100 {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func validKey(key string) bool {
	return key != "" && len(key) <= 1024 && !strings.HasPrefix(key, ".repo3/")
}

func md5Hex(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}
