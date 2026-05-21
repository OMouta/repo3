package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestClientHTTPMapping(t *testing.T) {
	var requests []string
	var putContentType string
	var putOwner string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		if r.URL.Query().Get("list-type") == "2" {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<ListBucketResult><Contents><Key>images/cat.txt</Key><LastModified>2026-05-21T14:32:00Z</LastModified><ETag>"abc123"</ETag><Size>3</Size></Contents></ListBucketResult>`))
			return
		}
		switch r.URL.Path {
		case "/memes":
			w.WriteHeader(http.StatusOK)
		case "/memes/images/cat.txt":
			switch r.Method {
			case http.MethodGet:
				_, _ = w.Write([]byte("cat"))
			case http.MethodPut:
				putContentType = r.Header.Get("Content-Type")
				putOwner = r.Header.Get("x-amz-meta-owner")
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusOK)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTP: server.Client()}
	ctx := context.Background()

	if err := c.CreateBucket(ctx, "memes"); err != nil {
		t.Fatal(err)
	}
	if err := c.PutObject(ctx, "memes", "images/cat.txt", strings.NewReader("cat"), 3, PutOptions{
		ContentType:  "text/plain",
		UserMetadata: map[string]string{"owner": "idk"},
	}); err != nil {
		t.Fatal(err)
	}
	resp, err := c.GetObject(ctx, "memes", "images/cat.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	closeIgnore(resp.Body)
	if err := c.DeleteObject(ctx, "memes", "images/cat.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListObjects(ctx, "memes", "images/"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"PUT /memes",
		"PUT /memes/images/cat.txt",
		"GET /memes/images/cat.txt",
		"DELETE /memes/images/cat.txt",
		"GET /memes?list-type=2&prefix=images%2F",
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests = %#v, want %#v", requests, want)
	}
	if putContentType != "text/plain" {
		t.Fatalf("Content-Type = %q", putContentType)
	}
	if putOwner != "idk" {
		t.Fatalf("x-amz-meta-owner = %q", putOwner)
	}
}
