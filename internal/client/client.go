package client

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Client struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	HTTP      *http.Client
}

type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
}

type PutOptions struct {
	ContentType  string
	UserMetadata map[string]string
}

func (c *Client) CreateBucket(ctx context.Context, bucket string) error {
	req, err := c.newRequest(ctx, http.MethodPut, "/"+escapePath(bucket), nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errorFromResponse(resp)
	}
	return resp.Body.Close()
}

func (c *Client) PutObject(ctx context.Context, bucket string, key string, body io.Reader, size int64, opts PutOptions) error {
	req, err := c.newRequest(ctx, http.MethodPut, "/"+escapePath(bucket)+"/"+escapePath(key), body)
	if err != nil {
		return err
	}
	req.ContentLength = size
	if opts.ContentType != "" {
		req.Header.Set("Content-Type", opts.ContentType)
	}
	for key, value := range opts.UserMetadata {
		req.Header.Set("x-amz-meta-"+key, value)
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errorFromResponse(resp)
	}
	return resp.Body.Close()
}

func (c *Client) GetObject(ctx context.Context, bucket string, key string) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/"+escapePath(bucket)+"/"+escapePath(key), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errorFromResponse(resp)
	}
	return resp, nil
}

func (c *Client) DeleteObject(ctx context.Context, bucket string, key string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, "/"+escapePath(bucket)+"/"+escapePath(key), nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errorFromResponse(resp)
	}
	return resp.Body.Close()
}

func (c *Client) ListObjects(ctx context.Context, bucket string, prefix string) ([]ObjectInfo, error) {
	q := url.Values{}
	q.Set("list-type", "2")
	if prefix != "" {
		q.Set("prefix", prefix)
	}
	req, err := c.newRequest(ctx, http.MethodGet, "/"+escapePath(bucket)+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errorFromResponse(resp)
	}
	defer closeIgnore(resp.Body)

	var parsed struct {
		Contents []struct {
			Key          string `xml:"Key"`
			LastModified string `xml:"LastModified"`
			ETag         string `xml:"ETag"`
			Size         int64  `xml:"Size"`
		} `xml:"Contents"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	out := make([]ObjectInfo, 0, len(parsed.Contents))
	for _, item := range parsed.Contents {
		lastModified, _ := time.Parse(time.RFC3339, item.LastModified)
		out = append(out, ObjectInfo{
			Key:          item.Key,
			Size:         item.Size,
			LastModified: lastModified,
			ETag:         strings.Trim(item.ETag, `"`),
		})
	}
	return out, nil
}

func (c *Client) newRequest(ctx context.Context, method, target string, body io.Reader) (*http.Request, error) {
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:9000"
	}
	base, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	rel, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	u := base.ResolveReference(rel)
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if c.AccessKey != "" {
		req.Header.Set("X-Repo3-Access-Key", c.AccessKey)
	}
	if c.SecretKey != "" {
		req.Header.Set("X-Repo3-Secret-Key", c.SecretKey)
	}
	return req, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return httpClient.Do(req)
}

func escapePath(p string) string {
	cleaned := path.Clean("/" + p)
	if cleaned == "/" {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(cleaned, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
