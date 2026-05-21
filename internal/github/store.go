package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"repo3/internal/storage"
)

const apiBase = "https://api.github.com"

type Store struct {
	token  string
	owner  string
	branch string
	client *http.Client
}

func NewStore(token, owner, branch string) *Store {
	if branch == "" {
		branch = "main"
	}
	return &Store{
		token:  token,
		owner:  owner,
		branch: branch,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Store) ListBuckets(ctx context.Context) ([]storage.Bucket, error) {
	var repos []repoResponse
	if err := s.doJSON(ctx, http.MethodGet, apiBase+"/orgs/"+url.PathEscape(s.owner)+"/repos?per_page=100&type=all", nil, &repos); err != nil {
		if !errors.Is(err, storage.ErrNoSuchBucket) {
			return nil, err
		}
		if err := s.doJSON(ctx, http.MethodGet, apiBase+"/users/"+url.PathEscape(s.owner)+"/repos?per_page=100&type=all", nil, &repos); err != nil {
			return nil, err
		}
	}

	buckets := make([]storage.Bucket, 0, len(repos))
	for _, repo := range repos {
		buckets = append(buckets, storage.Bucket{Name: repo.Name, CreationDate: repo.CreatedAt})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Name < buckets[j].Name })
	return buckets, nil
}

func (s *Store) CreateBucket(ctx context.Context, name string) error {
	body := map[string]any{"name": name, "auto_init": true}
	var out repoResponse
	err := s.doJSON(ctx, http.MethodPost, apiBase+"/orgs/"+url.PathEscape(s.owner)+"/repos", body, &out)
	if err == nil {
		return nil
	}
	if !errors.Is(err, storage.ErrNoSuchBucket) {
		return err
	}
	return s.doJSON(ctx, http.MethodPost, apiBase+"/user/repos", body, &out)
}

func (s *Store) DeleteBucket(ctx context.Context, name string) error {
	return storage.ErrBucketNotEmpty
}

func (s *Store) PutObject(ctx context.Context, bucket, key string, body io.Reader, meta storage.Metadata) (*storage.PutResult, error) {
	if strings.HasPrefix(key, ".repo3/") || key == "" {
		return nil, storage.ErrInvalidKey
	}

	data, err := io.ReadAll(io.LimitReader(body, 10*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 10*1024*1024 {
		return nil, fmt.Errorf("object too large")
	}

	existing, err := s.getContent(ctx, bucket, key, s.branch)
	if err != nil && !errors.Is(err, storage.ErrNoSuchKey) && !errors.Is(err, storage.ErrNoSuchBucket) {
		return nil, err
	}
	if errors.Is(err, storage.ErrNoSuchBucket) {
		return nil, err
	}

	reqBody := putContentRequest{
		Message: commitMessage("put", bucket, key, meta),
		Content: base64.StdEncoding.EncodeToString(data),
		Branch:  s.branch,
	}
	if existing.SHA != "" {
		reqBody.SHA = existing.SHA
	}

	var out writeContentResponse
	if err := s.doJSON(ctx, http.MethodPut, s.contentsURL(bucket, key), reqBody, &out); err != nil {
		return nil, err
	}
	sidecar := objectMetadata{
		ContentType:  meta.ContentType,
		ETag:         out.Content.SHA,
		UserMetadata: meta.UserMetadata,
	}
	if err := s.putMetadata(ctx, bucket, key, sidecar); err != nil {
		return nil, err
	}
	return &storage.PutResult{ETag: out.Content.SHA, VersionID: out.Commit.SHA}, nil
}

func (s *Store) GetObject(ctx context.Context, bucket, key string, versionID string) (*storage.Object, error) {
	ref := s.branch
	if versionID != "" {
		ref = versionID
	}

	content, err := s.getContent(ctx, bucket, key, ref)
	if err != nil {
		return nil, err
	}

	info := storage.ObjectInfo{
		Bucket:       bucket,
		Key:          key,
		Size:         content.Size,
		ETag:         content.SHA,
		LastModified: time.Now().UTC(),
		VersionID:    versionID,
	}
	if meta, err := s.readMetadata(ctx, bucket, key, ref); err == nil {
		info.ContentType = meta.ContentType
		if meta.ETag != "" {
			info.ETag = meta.ETag
		}
		info.UserMetadata = meta.UserMetadata
	}

	if content.DownloadURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, content.DownloadURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			return nil, mapStatus(resp.StatusCode, data)
		}
		if resp.Header.Get("Content-Type") != "" {
			// Prefer explicit Repo3 metadata when present.
			if info.ContentType == "" {
				info.ContentType = resp.Header.Get("Content-Type")
			}
		}
		return &storage.Object{Info: info, Body: resp.Body}, nil
	}

	data, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content.Content, "\n", ""))
	if err != nil {
		return nil, err
	}
	info.Size = int64(len(data))
	return &storage.Object{Info: info, Body: io.NopCloser(bytes.NewReader(data))}, nil
}

func (s *Store) HeadObject(ctx context.Context, bucket, key string, versionID string) (*storage.ObjectInfo, error) {
	obj, err := s.GetObject(ctx, bucket, key, versionID)
	if err != nil {
		return nil, err
	}
	defer obj.Body.Close()
	return &obj.Info, nil
}

func (s *Store) DeleteObject(ctx context.Context, bucket, key string) error {
	content, err := s.getContent(ctx, bucket, key, s.branch)
	if err != nil {
		return err
	}

	reqBody := deleteContentRequest{
		Message: fmt.Sprintf("repo3: delete object %s", key),
		SHA:     content.SHA,
		Branch:  s.branch,
	}
	var out writeContentResponse
	if err := s.doJSON(ctx, http.MethodDelete, s.contentsURL(bucket, key), reqBody, &out); err != nil {
		return err
	}
	if err := s.deleteMetadata(ctx, bucket, key); err != nil && !errors.Is(err, storage.ErrNoSuchKey) {
		return err
	}
	return nil
}

func (s *Store) ListObjects(ctx context.Context, bucket, prefix string, limit int) ([]storage.ObjectInfo, error) {
	var tree treeResponse
	u := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1", apiBase, url.PathEscape(s.owner), url.PathEscape(bucket), url.PathEscape(s.branch))
	if err := s.doJSON(ctx, http.MethodGet, u, nil, &tree); err != nil {
		return nil, err
	}

	out := make([]storage.ObjectInfo, 0)
	for _, item := range tree.Tree {
		if item.Type != "blob" || !strings.HasPrefix(item.Path, prefix) || strings.HasPrefix(item.Path, ".repo3/") {
			continue
		}
		out = append(out, storage.ObjectInfo{
			Bucket: bucket,
			Key:    item.Path,
			Size:   item.Size,
			ETag:   item.SHA,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Store) getContent(ctx context.Context, bucket, key, ref string) (contentResponse, error) {
	var out contentResponse
	u := s.contentsURL(bucket, key)
	if ref != "" {
		u += "?ref=" + url.QueryEscape(ref)
	}
	if err := s.doJSON(ctx, http.MethodGet, u, nil, &out); err != nil {
		if errors.Is(err, storage.ErrNoSuchBucket) {
			if s.repoExists(ctx, bucket) {
				return contentResponse{}, storage.ErrNoSuchKey
			}
		}
		return contentResponse{}, err
	}
	if out.Type != "" && out.Type != "file" {
		return contentResponse{}, storage.ErrInvalidKey
	}
	return out, nil
}

func (s *Store) repoExists(ctx context.Context, bucket string) bool {
	var out repoResponse
	u := fmt.Sprintf("%s/repos/%s/%s", apiBase, url.PathEscape(s.owner), url.PathEscape(bucket))
	return s.doJSON(ctx, http.MethodGet, u, nil, &out) == nil
}

func (s *Store) contentsURL(bucket, key string) string {
	return fmt.Sprintf("%s/repos/%s/%s/contents/%s", apiBase, url.PathEscape(s.owner), url.PathEscape(bucket), escapePath(key))
}

func (s *Store) putMetadata(ctx context.Context, bucket, key string, meta objectMetadata) error {
	if meta.ContentType == "" && meta.ETag == "" && len(meta.UserMetadata) == 0 {
		return nil
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	sidecarKey := metadataKey(key)
	existing, err := s.getContent(ctx, bucket, sidecarKey, s.branch)
	if err != nil && !errors.Is(err, storage.ErrNoSuchKey) {
		return err
	}

	reqBody := putContentRequest{
		Message: fmt.Sprintf("repo3: put metadata %s", key),
		Content: base64.StdEncoding.EncodeToString(append(data, '\n')),
		Branch:  s.branch,
	}
	if existing.SHA != "" {
		reqBody.SHA = existing.SHA
	}
	var out writeContentResponse
	return s.doJSON(ctx, http.MethodPut, s.contentsURL(bucket, sidecarKey), reqBody, &out)
}

func (s *Store) readMetadata(ctx context.Context, bucket, key, ref string) (objectMetadata, error) {
	content, err := s.getContent(ctx, bucket, metadataKey(key), ref)
	if err != nil {
		return objectMetadata{}, err
	}
	data, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content.Content, "\n", ""))
	if err != nil {
		return objectMetadata{}, err
	}
	var meta objectMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return objectMetadata{}, err
	}
	return meta, nil
}

func (s *Store) deleteMetadata(ctx context.Context, bucket, key string) error {
	sidecarKey := metadataKey(key)
	content, err := s.getContent(ctx, bucket, sidecarKey, s.branch)
	if err != nil {
		return err
	}

	reqBody := deleteContentRequest{
		Message: fmt.Sprintf("repo3: delete metadata %s", key),
		SHA:     content.SHA,
		Branch:  s.branch,
	}
	var out writeContentResponse
	return s.doJSON(ctx, http.MethodDelete, s.contentsURL(bucket, sidecarKey), reqBody, &out)
}

func metadataKey(key string) string {
	return ".repo3/meta/" + strings.TrimPrefix(key, "/") + ".json"
}

func (s *Store) doJSON(ctx context.Context, method, url string, in any, out any) error {
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return mapStatus(resp.StatusCode, data)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func mapStatus(status int, body []byte) error {
	message := githubMessage(body)
	switch status {
	case http.StatusNotFound:
		return storage.ErrNoSuchBucket
	case http.StatusConflict, http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: %s", storage.ErrOperationAborted, messageOrDefault(message, "GitHub rejected the write."))
	case http.StatusForbidden, http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", storage.ErrAccessDenied, messageOrDefault(message, "GitHub rejected the token or repository permission."))
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", storage.ErrSlowDown, messageOrDefault(message, "GitHub rate limit exceeded."))
	default:
		return fmt.Errorf("github request failed with status %d: %s", status, message)
	}
}

func githubMessage(body []byte) string {
	var parsed struct {
		Message string `json:"message"`
	}
	if len(body) == 0 || json.Unmarshal(body, &parsed) != nil {
		return strings.TrimSpace(string(body))
	}
	return parsed.Message
}

func messageOrDefault(message, fallback string) string {
	if strings.TrimSpace(message) == "" {
		return fallback
	}
	return message
}

func commitMessage(action, bucket, key string, meta storage.Metadata) string {
	var b strings.Builder
	fmt.Fprintf(&b, "repo3: %s object %s\n\nBucket: %s\nKey: %s", action, key, bucket, key)
	if meta.ContentType != "" {
		fmt.Fprintf(&b, "\nContent-Type: %s", meta.ContentType)
	}
	if len(meta.UserMetadata) > 0 {
		keys := make([]string, 0, len(meta.UserMetadata))
		for key := range meta.UserMetadata {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&b, "\nMetadata-%s: %s", key, meta.UserMetadata[key])
		}
	}
	return b.String()
}

func escapePath(p string) string {
	cleaned := path.Clean("/" + p)
	parts := strings.Split(strings.TrimPrefix(cleaned, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

type repoResponse struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type contentResponse struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int64  `json:"size"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	DownloadURL string `json:"download_url"`
}

type putContentRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	SHA     string `json:"sha,omitempty"`
	Branch  string `json:"branch,omitempty"`
}

type deleteContentRequest struct {
	Message string `json:"message"`
	SHA     string `json:"sha"`
	Branch  string `json:"branch,omitempty"`
}

type writeContentResponse struct {
	Content contentResponse `json:"content"`
	Commit  struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

type treeResponse struct {
	Tree []treeItem `json:"tree"`
}

type treeItem struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	Size int64  `json:"size"`
}

type objectMetadata struct {
	ContentType  string            `json:"contentType,omitempty"`
	ETag         string            `json:"etag,omitempty"`
	UserMetadata map[string]string `json:"userMetadata,omitempty"`
}
